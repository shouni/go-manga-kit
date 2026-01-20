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
	imagePrompt   prompts.ImagePrompt
	mangaComposer *generator.MangaComposer
}

// New は、New は、設定とキャラクター定義を基に新しい Manager を初期化します。
func New(ctx context.Context, cfg config.Config, httpClient httpkit.ClientInterface, ioFactory remoteio.IOFactory, imagePrompt prompts.ImagePrompt, charData []byte) (*Manager, error) {
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

	prompt, err := initializeImagePrompt(imagePrompt, charData, cfg.StyleSuffix)
	if err != nil {
		return nil, err
	}

	mangaComposer, err := buildMangaComposer(cfg, httpClient, aiClient, reader, charData)
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

// initializeImagePrompt は ImagePromptBuilderを初期化します。
func initializeImagePrompt(imagePrompt prompts.ImagePrompt, charData []byte, styleSuffix string) (prompts.ImagePrompt, error) {
	chars, err := domain.GetCharacters(charData)
	if err != nil {
		return nil, fmt.Errorf("failed to get characters from data: %w", err)
	}

	pb := imagePrompt
	if pb == nil {
		pb = prompts.NewImagePromptBuilder(chars, styleSuffix)
	}

	return pb, nil
}
