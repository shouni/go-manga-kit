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

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"

	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/gcsfactory"
)

// ExecuteImageOnly は、指定されたJSONファイル（台本）を読み込み、
// 画像生成と公開処理（Phase 2 & 3）を実行するのだ。
func ExecuteImageOnly(ctx context.Context, cfg *config.Config) error {
	appCtx, err := setupAppContext(ctx, cfg)
	if err != nil {
		return err
	}

	// JSONファイルの読み込み
	rc, err := appCtx.Reader.Open(ctx, cfg.Options.ScriptFile)
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

	rc, err := appCtx.Reader.Open(ctx, cfg.Options.ScriptFile)
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

	outputPath := cfg.Options.OutputFile
	err = appCtx.Writer.Write(ctx, outputPath, bytes.NewReader(resp.Data), resp.MimeType)
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
	aiClient, err := builder.InitializeAIClient(ctx, cfg.GeminiAPIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create ai client: %w", err)
	}

	gcsFactory, err := gcsfactory.NewGCSClientFactory(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client factory: %w", err)
	}

	reader, err := gcsFactory.NewInputReader()
	if err != nil {
		return nil, err
	}
	writer, err := gcsFactory.NewOutputWriter()
	if err != nil {
		return nil, err
	}

	// Pipelineを一度だけ初期化
	mangaPipeline, err := builder.InitializeMangaPipeline(httpClient, aiClient, cfg.GeminiModel, cfg.Options.ScriptFile)
	if err != nil {
		return nil, err
	}

	appCtx := builder.NewAppContext(cfg, httpClient, aiClient, reader, writer, *mangaPipeline)
	return &appCtx, nil
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
