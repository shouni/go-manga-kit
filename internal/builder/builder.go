package builder

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shouni/go-manga-kit/internal/runner"
	"github.com/shouni/go-manga-kit/pkg/domain"

	"github.com/patrickmn/go-cache"
	imagekit "github.com/shouni/gemini-image-kit/pkg/adapters"
	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-text-format/pkg/builder"
	"google.golang.org/genai"
)

// BuildImageRunner は個別パネル画像生成を担当する Runner を構築します。
func BuildImageRunner(ctx context.Context, appCtx *AppContext) (runner.ImageRunner, error) {
	// 共通の Core コンポーネントを作成
	imgCore := buildSharedImageCore(appCtx)

	// 個別パネル用アダプターの初期化
	imageAdapter, err := imagekit.NewGeminiImageAdapter(
		imgCore,
		appCtx.aiClient,
		appCtx.Config.GeminiImageModel,
		appCtx.Config.ImagePromptSuffix,
	)
	if err != nil {
		return nil, fmt.Errorf("画像アダプターの初期化に失敗しました: %w", err)
	}

	chars, err := domain.LoadCharacters(appCtx.Options.CharacterConfig)
	if err != nil {
		return nil, fmt.Errorf("キャラクター情報の取得に失敗しました: %w", err)
	}

	return runner.NewMangaImageRunner(
		imageAdapter,
		chars,
		appCtx.Options.PanelLimit,
		appCtx.Config.ImagePromptSuffix,
	), nil
}

// BuildMangaPageRunner は 8パネル一括のページ生成を担当する Runner を構築します。
func BuildMangaPageRunner(ctx context.Context, appCtx *AppContext) (*runner.MangaPageRunner, error) {
	// 共通の Core コンポーネントを作成
	imgCore := buildSharedImageCore(appCtx)

	// Adapterの生成
	pageAdapter := imagekit.NewGeminiMangaPageAdapter(
		imgCore,
		appCtx.aiClient,
		appCtx.Config.GeminiModel, // 使用するモデル名
	)
	// TODO:: 参照パッケージ側の修正を確認
	//if err != nil {
	//	return nil, fmt.Errorf("Adapterの初期化に失敗したのだ: %w", err)
	//}

	// 1. キャラクター情報の取得 (pkg/domain/character.go の LoadCharacters を使用)
	// この戻り値は map[string]domain.Character なのだ！
	chars, err := domain.LoadCharacters(appCtx.Options.CharacterConfig)
	if err != nil {
		return nil, fmt.Errorf("キャラクター情報の取得に失敗したのだ: %w", err)
	}

	// 2. Runner の生成
	// ポインタへの変換は不要になったので、chars をそのまま渡せるのだ。
	// 第3引数は、Runner内でPagePipelineに渡されるスタイル指定なのだ。
	return runner.NewMangaPageRunner(
		*pageAdapter,
		chars,
		appCtx.Config.ImagePromptSuffix,
		appCtx.Options.ScriptFile,
	), nil
}

// buildSharedImageCore は各アダプターで共有する画像処理コアを生成します。
func buildSharedImageCore(appCtx *AppContext) *imagekit.GeminiImageCore {
	// 参照画像のダウンロード結果を保持するキャッシュ
	imgCache := cache.New(30*time.Minute, 1*time.Hour)
	cacheTTL := 1 * time.Hour

	return imagekit.NewGeminiImageCore(
		appCtx.httpClient,
		imgCache,
		cacheTTL,
	)
}

// BuildPublisherRunner はコンテンツ保存と変換を行う Runner を構築します。
func BuildPublisherRunner(ctx context.Context, appCtx *AppContext) (runner.PublisherRunner, error) {
	opts := appCtx.Options

	gcsClient := appCtx.RemoteIOFactory
	writer, err := gcsClient.NewOutputWriter()
	if err != nil {
		slog.WarnContext(ctx, "OutputWriterの取得に失敗しました。保存機能が制限される可能性があります", "error", err)
	}

	config := builder.BuilderConfig{
		EnableHardWraps: true,
		Mode:            "webtoon",
	}
	appBuilder, err := builder.NewBuilder(config)
	if err != nil {
		return nil, fmt.Errorf("アプリケーションビルダーの初期化に失敗しました: %w", err)
	}

	md2htmlRunner, err := appBuilder.BuildRunner()
	if err != nil {
		return nil, fmt.Errorf("MarkdownToHtmlRunnerの初期化に失敗しました: %w", err)
	}

	return runner.NewDefaultPublisherRunner(opts, writer, md2htmlRunner), nil
}

// InitializeAIClient は gemini クライアントを初期化します。
func InitializeAIClient(ctx context.Context, apiKey string) (gemini.GenerativeModel, error) {
	const defaultGeminiTemperature = float32(0.2)
	clientConfig := gemini.Config{
		APIKey:      apiKey,
		Temperature: genai.Ptr(defaultGeminiTemperature),
	}
	aiClient, err := gemini.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("AIクライアントの初期化に失敗しました: %w", err)
	}
	return aiClient, nil
}
