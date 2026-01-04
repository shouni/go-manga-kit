package builder

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/shouni/go-manga-kit/internal/runner"
	"github.com/shouni/go-manga-kit/pkg/domain"
	mngkit "github.com/shouni/go-manga-kit/pkg/pipeline"

	"github.com/shouni/gemini-image-kit/pkg/generator"
	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-text-format/pkg/builder"
	"google.golang.org/genai"
)

// BuildImageRunner は個別パネル画像生成を担当する Runner を構築します。
func BuildImageRunner(ctx context.Context, appCtx *AppContext) (runner.ImageRunner, error) {
	imgGen, err := InitializeImageGenerator(appCtx)
	if err != nil {
		return nil, fmt.Errorf("GeminiGeneratorの初期化に失敗したのだ: %w", err)
	}

	chars, err := domain.LoadCharacters(appCtx.Options.CharacterConfig)
	if err != nil {
		return nil, fmt.Errorf("キャラクター情報の取得に失敗しました: %w", err)
	}

	return runner.NewMangaImageRunner(
		imgGen,
		chars,
		appCtx.Options.PanelLimit,
		appCtx.Config.ImagePromptSuffix,
	), nil
}

// BuildMangaPageRunner は 8パネル一括のページ生成を担当する Runner を構築します。
func BuildMangaPageRunner(ctx context.Context, appCtx *AppContext) (*runner.MangaPageRunner, error) {
	imgGen, err := InitializeImageGenerator(appCtx)
	if err != nil {
		return nil, fmt.Errorf("GeminiGeneratorの初期化に失敗したのだ: %w", err)
	}

	chars, err := domain.LoadCharacters(appCtx.Options.CharacterConfig)
	if err != nil {
		return nil, fmt.Errorf("キャラクター情報の取得に失敗しました: %w", err)
	}

	// 2. Runner の生成
	// ポインタへの変換は不要になったので、chars をそのまま渡せるのだ。
	// 第3引数は、Runner内でPagePipelineに渡されるスタイル指定なのだ。
	return runner.NewMangaPageRunner(
		imgGen,
		chars,
		appCtx.Config.ImagePromptSuffix,
		appCtx.Options.ScriptFile,
	), nil
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

// InitializeImageGenerator は ImageGeneratorを初期化します。
func InitializeImageGenerator(appCtx *AppContext) (generator.ImageGenerator, error) {
	imgGen, err := mngkit.InitializeImageGenerator(appCtx.httpClient, appCtx.aiClient, appCtx.Config.GeminiModel)
	if err != nil {
		return nil, fmt.Errorf("GeminiGeneratorの初期化に失敗したのだ: %w", err)
	}

	return imgGen, nil
}
