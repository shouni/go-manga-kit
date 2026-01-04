package builder

import (
	"context"
	"fmt"

	"github.com/shouni/go-manga-kit/internal/runner"
	mngkit "github.com/shouni/go-manga-kit/pkg/pipeline"

	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-text-format/pkg/builder"
	"google.golang.org/genai"
)

// BuildImageRunner は個別パネル画像生成を担当する Runner を構築します。
func BuildImageRunner(ctx context.Context, appCtx *AppContext) (runner.ImageRunner, error) {
	manga, err := initializeMangaPipeline(appCtx)
	if err != nil {
		return nil, fmt.Errorf("キャラクター情報の取得に失敗しました: %w", err)
	}

	return runner.NewMangaImageRunner(
		*manga,
		appCtx.Options.PanelLimit,
		appCtx.Config.ImagePromptSuffix,
	), nil
}

// BuildMangaPageRunner は 8パネル一括のページ生成を担当する Runner を構築します。
func BuildMangaPageRunner(ctx context.Context, appCtx *AppContext) (*runner.MangaPageRunner, error) {
	manga, err := initializeMangaPipeline(appCtx)
	if err != nil {
		return nil, fmt.Errorf("キャラクター情報の取得に失敗しました: %w", err)
	}

	// 2. Runner の生成
	return runner.NewMangaPageRunner(
		*manga,
		appCtx.Config.ImagePromptSuffix,
		appCtx.Options.ScriptFile,
	), nil
}

// BuildPublisherRunner はコンテンツ保存と変換を行う Runner を構築します。
func BuildPublisherRunner(ctx context.Context, appCtx *AppContext) (runner.PublisherRunner, error) {
	opts := appCtx.Options
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

	return runner.NewDefaultPublisherRunner(opts, appCtx.Writer, md2htmlRunner), nil
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

// initializeMangaPipeline は MangaPipelineを初期化します。
func initializeMangaPipeline(appCtx *AppContext) (*mngkit.Pipeline, error) {
	pl, err := mngkit.NewPipeline(appCtx.httpClient, appCtx.aiClient, appCtx.Config.GeminiImageModel, appCtx.Options.ScriptFile)
	if err != nil {
		return nil, fmt.Errorf("GeminiGeneratorの初期化に失敗しました: %w", err)
	}

	return pl, nil
}
