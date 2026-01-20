package workflow

import (
	"context"
	"fmt"

	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-manga-kit/pkg/prompts"

	"github.com/shouni/go-gemini-client/pkg/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"google.golang.org/genai"
)

// Manager は、ワークフローの各工程を担う Runner 群を構築・管理します。
type Manager struct {
	cfg           config.Config
	httpClient    httpkit.ClientInterface
	reader        remoteio.InputReader
	writer        remoteio.OutputWriter
	aiClient      gemini.GenerativeModel
	scriptPrompt  prompts.ScriptPrompt
	imagePrompt   prompts.ImagePrompt
	mangaComposer *generator.MangaComposer
}

// New は、New は、設定とキャラクター定義を基に新しい Manager を初期化します。
func New(ctx context.Context, cfg config.Config, httpClient httpkit.ClientInterface, ioFactory remoteio.IOFactory, scriptPrompt prompts.ScriptPrompt, imagePrompt prompts.ImagePrompt, charData []byte) (*Manager, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("httpClient は必須です")
	}
	if ioFactory == nil {
		return nil, fmt.Errorf("IOFactory は必須です")
	}

	reader, _ := ioFactory.InputReader()
	writer, _ := ioFactory.OutputWriter()

	aiClient, err := initializeAIClient(ctx, cfg.GeminiAPIKey)
	if err != nil {
		return nil, err
	}

	// charDataをここで一度だけパースする
	chars, err := domain.GetCharacters(charData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse character data: %w", err)
	}

	prompt, err := initializeImagePrompt(imagePrompt, chars, cfg.StyleSuffix)
	if err != nil {
		return nil, err
	}

	mangaComposer, err := buildMangaComposer(cfg, httpClient, aiClient, reader, chars)
	if err != nil {
		return nil, fmt.Errorf("画像生成エンジンの初期化に失敗しました: %w", err)
	}

	return &Manager{
		cfg:           cfg,
		httpClient:    httpClient,
		reader:        reader,
		writer:        writer,
		aiClient:      aiClient,
		imagePrompt:   prompt,
		mangaComposer: mangaComposer,
	}, nil
}

// initializeAIClient は gemini クライアントを初期化します。
func initializeAIClient(ctx context.Context, apiKey string) (gemini.GenerativeModel, error) {
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

// initializeScriptPrompt は ScriptPrompt ビルダーを初期化します。
// 引数として既存のビルダーが渡された場合はそれを返し、nil の場合は新規作成します。
func initializeScriptPrompt(scriptPrompt prompts.ScriptPrompt) (prompts.ScriptPrompt, error) {
	if scriptPrompt != nil {
		return scriptPrompt, nil
	}

	pb, err := prompts.NewTextPromptBuilder()
	if err != nil {
		return nil, fmt.Errorf("TextPromptBuilder の新規作成に失敗しました: %w", err)
	}

	return pb, nil
}

// initializeImagePrompt は ImagePromptBuilderを初期化します。
// 引数として既存のビルダーが渡された場合はそれを返し、nil の場合は新規作成します。
func initializeImagePrompt(imagePrompt prompts.ImagePrompt, chars domain.CharactersMap, styleSuffix string) (prompts.ImagePrompt, error) {
	if imagePrompt != nil {
		return imagePrompt, nil
	}

	return prompts.NewImagePromptBuilder(chars, styleSuffix), nil
}
