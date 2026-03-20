package workflow

import (
	"fmt"

	"github.com/shouni/go-gemini-client/pkg/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/remoteio"

	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-manga-kit/pkg/ports"
)

// PromptDependencies はプロンプト関連の依存関係をまとめた構造体です。
type PromptDependencies struct {
	CharactersMap ports.CharactersMap
	ScriptPrompt  ports.ScriptPrompt
	ImagePrompt   ports.ImagePrompt
}

type ManagerArgs struct {
	Config             config.Config
	HTTPClient         httpkit.HTTPClient
	Reader             remoteio.InputReader
	Writer             remoteio.OutputWriter
	AIClient           gemini.GenerativeModel
	PromptDependencies *PromptDependencies
}

// Manager は、ワークフローの各工程を担う Runner 群を構築・管理します。
type Manager struct {
	cfg                config.Config
	httpClient         httpkit.HTTPClient
	reader             remoteio.InputReader
	writer             remoteio.OutputWriter
	aiClient           gemini.GenerativeModel
	mangaComposer      *generator.MangaComposer
	promptDependencies *PromptDependencies
}

// New は、設定とキャラクター定義を基に新しい Manager を初期化します。
func New(args ManagerArgs) (*ports.Workflows, error) {
	if err := validateArgs(&args); err != nil {
		return nil, err
	}

	cfg := args.Config
	cfg.ApplyDefaults()

	m := &Manager{
		cfg:                cfg,
		httpClient:         args.HTTPClient,
		reader:             args.Reader,
		writer:             args.Writer,
		aiClient:           args.AIClient,
		promptDependencies: args.PromptDependencies,
	}

	var err error
	// validateArgs で nil チェック済みのため、安全にアクセス可能
	m.mangaComposer, err = m.buildMangaComposer(args.PromptDependencies.CharactersMap)
	if err != nil {
		return nil, err
	}

	// 内部で全ての Runner インスタンスを生成して返す
	runners, err := m.buildAllRunners()
	if err != nil {
		return nil, err
	}

	return runners, nil
}

// validateArgs は引数のバリデーションを行います。
func validateArgs(args *ManagerArgs) error {
	if args.HTTPClient == nil {
		return fmt.Errorf("HTTPClient is required")
	}
	if args.Reader == nil {
		return fmt.Errorf("InputReader is required")
	}
	if args.Writer == nil {
		return fmt.Errorf("OutputWriter is required")
	}
	if args.AIClient == nil {
		return fmt.Errorf("AIClient is required")
	}
	if args.PromptDependencies == nil {
		return fmt.Errorf("PromptDependencies is required")
	}
	if args.PromptDependencies.CharactersMap == nil {
		return fmt.Errorf("CharactersMap is required")
	}
	if args.PromptDependencies.ScriptPrompt == nil {
		return fmt.Errorf("ScriptPrompt is required")
	}
	if args.PromptDependencies.ImagePrompt == nil {
		return fmt.Errorf("ImagePrompt is required")
	}

	return nil
}
