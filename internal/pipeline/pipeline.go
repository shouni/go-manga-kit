package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/shouni/go-manga-kit/internal/builder"
	"github.com/shouni/go-manga-kit/internal/config"
	"github.com/shouni/go-manga-kit/pkg/domain"

	//	imagekit "github.com/shouni/gemini-image-kit/pkg/adapters"
	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"

	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/gcsfactory"
)

// Execute は依存関係を初期化し、漫画生成パイプラインを3つのフェーズで実行するのだ。
func Execute(ctx context.Context, cfg *config.Config) error {
	// 設定のロードと依存関係の準備
	appCtx, err := setupAppContext(ctx, cfg)
	if err != nil {
		return err
	}

	// --- Phase 1: Script Phase (台本取得 & 生成) ---
	manga, err := runScriptStep(ctx, appCtx)
	if err != nil {
		return err
	}

	// --- Phase 2: Image Phase (イメージ作成) ---
	images, err := runImageStep(ctx, appCtx, manga)
	if err != nil {
		return err
	}

	// --- Phase 3: Publish Phase (公開/保存) ---
	err = runPublishStep(ctx, appCtx, manga, images)
	if err != nil {
		return err
	}

	slog.Info("全てのフェーズが正常に完了したのだ！")
	return nil
}

// ExecuteScriptOnly は台本の生成（Phase 1）のみを行い、結果をJSONファイルとして保存するのだ。
func ExecuteScriptOnly(ctx context.Context, cfg *config.Config) error {
	appCtx, err := setupAppContext(ctx, cfg)
	if err != nil {
		return err
	}

	// --- Phase 1: Script Phase (台本取得 & 生成) ---
	manga, err := runScriptStep(ctx, appCtx)
	if err != nil {
		return err
	}

	// JSONとしてファイル保存
	outputPath := cfg.Options.OutputFile
	// 保存処理 (簡易的に json.Marshal を使うのだ)
	data, err := json.MarshalIndent(manga, "", "  ")
	if err != nil {
		return fmt.Errorf("JSONの整形に失敗したのだ: %w", err)
	}

	// Writerを取得して書き込むのだ
	writer, err := appCtx.RemoteIOFactory.NewOutputWriter()
	if err != nil {
		return fmt.Errorf("出力ライターの初期化に失敗したのだ: %w", err)
	}
	err = writer.Write(ctx, outputPath, bytes.NewReader(data), "application/json")
	if err != nil {
		return fmt.Errorf("JSONの保存に失敗したのだ: %w", err)
	}

	slog.Info("台本JSONの出力が完了したのだ！", "path", outputPath)
	return nil
}

// ExecuteImageOnly は、指定されたJSONファイル（台本）を読み込み、
// 画像生成と公開処理（Phase 2 & 3）を実行するのだ。
func ExecuteImageOnly(ctx context.Context, cfg *config.Config) error {
	appCtx, err := setupAppContext(ctx, cfg)
	if err != nil {
		return err
	}

	// JSONファイルの読み込み
	reader, err := appCtx.RemoteIOFactory.NewInputReader()
	if err != nil {
		return err
	}
	rc, err := reader.Open(ctx, cfg.Options.ScriptFile)
	if err != nil {
		return fmt.Errorf("JSONファイル '%s' の読み込みに失敗しました: %w", cfg.Options.ScriptFile, err)
	}
	defer rc.Close()

	var manga domain.MangaResponse
	if err := json.NewDecoder(rc).Decode(&manga); err != nil {
		return fmt.Errorf("JSONファイル '%s' のデコードに失敗しました: %w", cfg.Options.ScriptFile, err)
	}

	// --- Phase 2: Image Phase (イメージ作成) ---
	images, err := runImageStep(ctx, appCtx, manga)
	if err != nil {
		return err
	}

	// --- Phase 3: Publish Phase (公開/保存) ---
	err = runPublishStep(ctx, appCtx, manga, images)
	if err != nil {
		return err
	}

	slog.Info("画像生成と公開処理が完了したのだ！")
	return nil
}

// ExecuteStoryOnly は、すでに Phase 2 & 3 で出力された台本ファイルを基に、
// MangaPageRunner を実行して「1枚の完成された漫画ページ」を生成する最終ステージなのだ！
func ExecuteStoryOnly(ctx context.Context, cfg *config.Config) error {
	appCtx, err := setupAppContext(ctx, cfg)
	if err != nil {
		return err
	}

	// 保存されている台本ファイル (Markdown) を読み込むのだ
	reader, err := appCtx.RemoteIOFactory.NewInputReader()
	if err != nil {
		return err
	}
	rc, err := reader.Open(ctx, cfg.Options.ScriptFile)
	if err != nil {
		return fmt.Errorf("台本ファイルの読み込みに失敗したのだ: %w", err)
	}
	defer rc.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, rc); err != nil {
		return err
	}
	markdownContent := buf.String()

	// MangaPageRunner を使って、一括生成の準備をするのだ
	pageRunner, err := builder.BuildMangaPageRunner(ctx, appCtx)
	if err != nil {
		return fmt.Errorf("MangaPageRunnerの構築に失敗したのだ: %w", err)
	}

	slog.Info("1枚の漫画ページとして一括生成を開始するのだ...")

	// 生成実行（Markdownをパースして統合プロンプトでAIに投げる
	resp, err := pageRunner.RunMarkdown(ctx, markdownContent)
	if err != nil {
		return fmt.Errorf("漫画ページの一括生成に失敗したのだ: %w", err)
	}

	// 最終成果物を保存するのだ、ここは単一の画像レスポンスなので、専用の保存処理を行うのだ
	writer, err := appCtx.RemoteIOFactory.NewOutputWriter()
	if err != nil {
		return err
	}

	outputPath := cfg.Options.OutputFile
	err = writer.Write(ctx, outputPath, bytes.NewReader(resp.Data), resp.MimeType)
	if err != nil {
		return fmt.Errorf("最終ページの保存に失敗したのだ: %w", err)
	}

	slog.Info("物語の集大成が完成したのだ！", "path", outputPath)
	return nil
}

// setupAppContext は、提供された設定と共有コンポーネントを使用して、アプリケーションコンテキストを初期化して返すのだ。
// ライフサイクル管理用の context と設定オブジェクトを受け取るのだ。
// 初期化中にエラーが発生した場合は、AppContext のポインタとエラーを返すのだ。
func setupAppContext(ctx context.Context, cfg *config.Config) (*builder.AppContext, error) {
	httpClient := httpkit.New(config.DefaultHTTPTimeout)
	gcsFactory, err := gcsfactory.NewGCSClientFactory(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client factory: %w", err)
	}
	aiClient, err := builder.InitializeAIClient(ctx, cfg.GeminiAPIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create ai client: %w", err)
	}

	appCtx := builder.NewAppContext(cfg, aiClient, httpClient, gcsFactory)
	return &appCtx, nil
}

// runScriptStep は ScriptRunner を使って台本(JSON)を生成するのだ
func runScriptStep(ctx context.Context, appCtx *builder.AppContext) (domain.MangaResponse, error) {
	slog.Info("Phase 1: 台本生成を開始するのだ...")
	scriptRunner, err := builder.BuildScriptRunner(ctx, appCtx)
	if err != nil {
		return domain.MangaResponse{}, fmt.Errorf("ScriptRunnerの構築に失敗したのだ: %w", err)
	}

	manga, err := scriptRunner.Run(ctx)
	if err != nil {
		return domain.MangaResponse{}, fmt.Errorf("台本生成に失敗したのだ: %w", err)
	}
	return manga, nil
}

// runImageStep は MangaImageRunner を使ってパネル画像を並列生成するのだ
func runImageStep(ctx context.Context, appCtx *builder.AppContext, manga domain.MangaResponse) ([]*imagedom.ImageResponse, error) {
	slog.Info("Phase 2: 画像生成を開始するのだ...", "pages", len(manga.Pages))
	imageRunner, err := builder.BuildImageRunner(ctx, appCtx)
	if err != nil {
		return nil, fmt.Errorf("ImageRunnerの構築に失敗したのだ: %w", err)
	}

	images, err := imageRunner.Run(ctx, manga)
	if err != nil {
		return nil, fmt.Errorf("画像生成に失敗したのだ: %w", err)
	}
	return images, nil
}

// runPublishStep は PublisherRunner を使って最終成果物を保存するのだ
func runPublishStep(ctx context.Context, appCtx *builder.AppContext, manga domain.MangaResponse, images []*imagedom.ImageResponse) error {
	slog.Info("Phase 3: 公開処理を開始するのだ...")
	publishRunner, err := builder.BuildPublisherRunner(ctx, appCtx)
	if err != nil {
		return fmt.Errorf("PublishRunnerの構築に失敗したのだ: %w", err)
	}

	err = publishRunner.Run(ctx, manga, images)
	if err != nil {
		return fmt.Errorf("公開処理に失敗したのだ: %w", err)
	}
	return nil
}
