package workflow

import (
	"fmt"

	"github.com/shouni/go-gemini-client/gemini"
	"github.com/shouni/go-http-kit/httpkit"
	"github.com/shouni/go-remote-io/remoteio"

	"github.com/shouni/go-manga-kit/layout"
	"github.com/shouni/go-manga-kit/ports"
)

// PromptDeps はプロンプト関連の依存関係をまとめた構造体です。
type PromptDeps struct {
	CharactersMap ports.CharactersMap
	ScriptPrompt  ports.ScriptPrompt
	ImagePrompt   ports.ImagePrompt
}

type ManagerArgs struct {
	Config     ports.Config
	HTTPClient httpkit.HTTPClient
	Reader     remoteio.InputReader
	Writer     remoteio.OutputWriter
	AIClient   gemini.GenerativeModel
	PromptDeps *PromptDeps
}

// manager は、ワークフローの各工程を担う Runner 群を構築・管理します。
type manager struct {
	cfg           ports.Config
	httpClient    httpkit.HTTPClient
	reader        remoteio.InputReader
	writer        remoteio.OutputWriter
	aiClient      gemini.GenerativeModel
	mangaComposer *layout.MangaComposer
	promptDeps    *PromptDeps
}

// NewWorkflows は、設定とキャラクター定義を基に新しい Workflows を初期化します。
func NewWorkflows(args ManagerArgs) (*ports.Workflows, error) {
	if err := validateArgs(&args); err != nil {
		return nil, err
	}

	cfg := args.Config
	cfg.ApplyDefaults()

	m := &manager{
		cfg:        cfg,
		httpClient: args.HTTPClient,
		reader:     args.Reader,
		writer:     args.Writer,
		aiClient:   args.AIClient,
		promptDeps: args.PromptDeps,
	}

	var err error
	// validateArgs で nil チェック済みのため、安全にアクセス可能
	m.mangaComposer, err = m.buildMangaComposer(args.PromptDeps.CharactersMap)
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
	if args.PromptDeps == nil {
		return fmt.Errorf("PromptDependencies is required")
	}
	if args.PromptDeps.CharactersMap == nil {
		return fmt.Errorf("CharactersMap is required")
	}
	if args.PromptDeps.ScriptPrompt == nil {
		return fmt.Errorf("ScriptPrompt is required")
	}
	if args.PromptDeps.ImagePrompt == nil {
		return fmt.Errorf("ImagePrompt is required")
	}

	return nil
}
