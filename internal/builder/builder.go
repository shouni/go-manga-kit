package builder

import (
	"context"
	"fmt"

	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-manga-kit/internal/runner"
	"github.com/shouni/go-manga-kit/pkg/parser"
	mngkit "github.com/shouni/go-manga-kit/pkg/pipeline"
	"github.com/shouni/go-manga-kit/pkg/publisher"

	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-text-format/pkg/builder"
	"google.golang.org/genai"
)

// BuildImageRunner は個別パネル画像生成を担当する MangaImageRunner を構築します。
func BuildImageRunner(ctx context.Context, appCtx *AppContext) (runner.ImageRunner, error) {
	return runner.NewMangaImageRunner(
		appCtx.MangaPipeline,
		appCtx.Config.ImagePromptSuffix,
		appCtx.Options.PanelLimit,
	), nil
}

// BuildMangaPageRunner は8パネル一括のページ生成を担当する MangaPageRunner を構築します。
func BuildMangaPageRunner(ctx context.Context, appCtx *AppContext) (runner.PageRunner, error) {
	// 2. Runner の生成
	return runner.NewMangaPageRunner(
		appCtx.MangaPipeline,
		appCtx.Config.ImagePromptSuffix,
		parser.NewParser(appCtx.Options.ScriptFile),
	), nil
}

// BuildPublisherRunner はコンテンツ保存と変換を行う Runner を構築します。
func BuildPublisherRunner(ctx context.Context, appCtx *AppContext) (runner.PublisherRunner, error) {
	opts := appCtx.Options
	config := builder.BuilderConfig{
		EnableHardWraps: true,
		Mode:            "webtoon",
	}
	md2htmlBuilder, err := builder.NewBuilder(config)
	if err != nil {
		return nil, fmt.Errorf("MarkdownToHtmlビルダーの初期化に失敗しました: %w", err)
	}

	md2htmlRunner, err := md2htmlBuilder.BuildRunner()
	if err != nil {
		return nil, fmt.Errorf("MarkdownToHtmlRunnerの初期化に失敗しました: %w", err)
	}
	pub := publisher.NewMangaPublisher(appCtx.Writer, md2htmlRunner)

	return runner.NewDefaultPublisherRunner(opts, pub), nil
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

// InitializeMangaPipeline は MangaPipelineを初期化します。
func InitializeMangaPipeline(httpClient httpkit.ClientInterface, aiClient gemini.GenerativeModel, model, characterConfig string) (mngkit.Pipeline, error) {
	pl, err := mngkit.NewPipeline(httpClient, aiClient, model, characterConfig)
	if err != nil {
		return mngkit.Pipeline{}, fmt.Errorf("GeminiGeneratorの初期化に失敗しました: %w", err)
	}

	return pl, nil
}
