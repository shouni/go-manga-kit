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
	aiClient      gemini.GenerativeModel
	reader        remoteio.InputReader
	writer        remoteio.OutputWriter
	mangaComposer *generator.MangaComposer
	imagePrompt   prompts.ImagePrompt
}

// New は、New は、設定とキャラクター定義を基に新しい Manager を初期化します。
func New(ctx context.Context, cfg config.Config, httpClient httpkit.ClientInterface, reader remoteio.InputReader, writer remoteio.OutputWriter, imagePrompt prompts.ImagePrompt, charData []byte) (*Manager, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("httpClient は必須です")
	}
	if reader == nil {
		return nil, fmt.Errorf("reader は必須です")
	}
	if writer == nil {
		return nil, fmt.Errorf("writer は必須です")
	}

	aiClient, err := initializeAIClient(ctx, cfg.GeminiAPIKey)
	if err != nil {
		return nil, err
	}

	mangaComposer, err := buildMangaComposer(cfg, httpClient, aiClient, reader, charData)
	if err != nil {
		return nil, fmt.Errorf("画像生成エンジンの初期化に失敗しました: %w", err)
	}

	prompt := imagePrompt
	if imagePrompt != nil {
		prompt, err = initializeImagePrompt(charData, cfg.StyleSuffix)
		if err != nil {
			return nil, fmt.Errorf("プロンプトマネージャーの初期化に失敗しました: %w", err)
		}
	}

	return &Manager{
		cfg:           cfg,
		httpClient:    httpClient,
		aiClient:      aiClient,
		reader:        reader,
		writer:        writer,
		mangaComposer: mangaComposer,
		imagePrompt:   prompt,
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
func initializeImagePrompt(charData []byte, styleSuffix string) (prompts.ImagePrompt, error) {
	chars, _ := domain.GetCharacters(charData)
	pb := prompts.NewImagePromptBuilder(chars, styleSuffix)

	return pb, nil
}
