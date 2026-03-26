package workflow

import (
	"fmt"

	imagePorts "github.com/shouni/gemini-image-kit/ports"
	"github.com/shouni/go-gemini-client/gemini"
	"github.com/shouni/go-http-kit/httpkit"
	"github.com/shouni/go-remote-io/remoteio"

	"github.com/shouni/go-manga-kit/layout"
	"github.com/shouni/go-manga-kit/ports"
)

type GenerationUnit struct {
	aiClient       gemini.GenerativeModel
	imageGenerator imagePorts.ImageGenerator
	mangaComposer  *layout.MangaComposer
	model          string
}

type LayoutManager struct {
	Standard *GenerationUnit
	Quality  *GenerationUnit
}

// PromptDeps はプロンプト関連の依存関係をまとめた構造体です。
type PromptDeps struct {
	CharactersMap ports.CharactersMap
	ScriptPrompt  ports.ScriptPrompt
	ImagePrompt   ports.ImagePrompt
}

type ManagerArgs struct {
	Config          ports.Config
	HTTPClient      httpkit.HTTPClient
	Reader          remoteio.InputReader
	Writer          remoteio.OutputWriter
	AIClient        gemini.GenerativeModel
	AIClientQuality gemini.GenerativeModel
	PromptDeps      *PromptDeps
}

// manager は、ワークフローの各工程を担う Runner 群を構築・管理します。
type manager struct {
	cfg             ports.Config
	httpClient      httpkit.HTTPClient
	reader          remoteio.InputReader
	writer          remoteio.OutputWriter
	aiClient        gemini.GenerativeModel
	aiClientQuality gemini.GenerativeModel
	layoutManager   LayoutManager
	promptDeps      *PromptDeps
}

// New は、設定とキャラクター定義を基に新しい Workflows を初期化します。
func New(args ManagerArgs) (*ports.Workflows, error) {
	if err := validateArgs(&args); err != nil {
		return nil, err
	}

	cfg := args.Config
	cfg.ApplyDefaults()

	m := &manager{
		cfg:             cfg,
		httpClient:      args.HTTPClient,
		reader:          args.Reader,
		writer:          args.Writer,
		aiClient:        args.AIClient,
		aiClientQuality: args.AIClientQuality,
		promptDeps:      args.PromptDeps,
	}

	var err error

	// --- Panel 用 LLM ユニットの構築 ---
	m.layoutManager.Standard, err = m.buildLLMUnit(m.aiClient, cfg.ImageStandardModel)
	if err != nil {
		return nil, fmt.Errorf("panel LLM unit の構築に失敗: %w", err)
	}

	// --- Page 用 LLM ユニットの構築 ---
	m.layoutManager.Quality, err = m.buildLLMUnit(m.aiClientQuality, cfg.ImageQualityModel)
	if err != nil {
		return nil, fmt.Errorf("page LLM unit の構築に失敗: %w", err)
	}

	return m.buildAllRunners()
}

// buildLLMUnit は、特定の AI クライアントとモデル設定に基づき、 core, composer, generator をひとまとめにした LLM 構造体を構築します。
func (m *manager) buildLLMUnit(client gemini.GenerativeModel, modelName string) (*GenerationUnit, error) {
	core, err := m.buildCore(client)
	if err != nil {
		return nil, err
	}

	composer, err := m.buildComposer(core, m.promptDeps.CharactersMap)
	if err != nil {
		return nil, err
	}

	gen, err := m.buildGenerator(core)
	if err != nil {
		return nil, err
	}

	return &GenerationUnit{
		aiClient:       client,
		imageGenerator: gen,
		mangaComposer:  composer,
		model:          modelName,
	}, nil
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
	if args.AIClientQuality == nil {
		// フォールバック
		args.AIClientQuality = args.AIClient
	}
	if args.PromptDeps == nil {
		return fmt.Errorf("PromptDeps is required")
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
