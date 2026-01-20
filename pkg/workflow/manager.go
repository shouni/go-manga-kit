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
func New(ctx context.Context, args ManagerArgs) (*Manager, error) {
	if args.HTTPClient == nil {
		return nil, fmt.Errorf("httpClient は必須です")
	}
	if args.Reader == nil {
		return nil, fmt.Errorf("InputReader は必須です")
	}
	if args.Writer == nil {
		return nil, fmt.Errorf("OutputWriter は必須です")
	}
	if args.CharactersMap == nil {
		return nil, fmt.Errorf("CharactersMap は必須です")
	}

	aiClient, err := initializeAIClient(ctx, args.Config.GeminiAPIKey)
	if err != nil {
		return nil, err
	}

	sPrompt, err := initializeScriptPrompt(args.ScriptPrompt)
	if err != nil {
		return nil, err
	}

	iPrompt, err := initializeImagePrompt(args.ImagePrompt, args.CharactersMap, args.Config.StyleSuffix)
	if err != nil {
		return nil, err
	}

	mangaComposer, err := buildMangaComposer(args.Config, args.HTTPClient, aiClient, args.Reader, args.CharactersMap)
	if err != nil {
		return nil, fmt.Errorf("画像生成エンジンの初期化に失敗しました: %w", err)
	}

	return &Manager{
		cfg:           args.Config,
		httpClient:    args.HTTPClient,
		reader:        args.Reader,
		writer:        args.Writer,
		aiClient:      aiClient,
		scriptPrompt:  sPrompt,
		imagePrompt:   iPrompt,
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
func initializeImagePrompt(imagePrompt prompts.ImagePrompt, charMap domain.CharactersMap, styleSuffix string) (prompts.ImagePrompt, error) {
	if imagePrompt != nil {
		return imagePrompt, nil
	}

	return prompts.NewImagePromptBuilder(charMap, styleSuffix), nil
}
