package workflow

import (
	"context"
	"fmt"

	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-manga-kit/pkg/runner"

	"github.com/shouni/go-gemini-client/pkg/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

type ManagerArgs struct {
	Config        config.Config
	HTTPClient    httpkit.HTTPClient
	Reader        remoteio.InputReader
	Writer        remoteio.OutputWriter
	CharactersMap domain.CharactersMap
	ScriptPrompt  domain.ScriptPrompt
	ImagePrompt   domain.ImagePrompt
	AIClient      gemini.GenerativeModel
}

// Manager は、ワークフローの各工程を担う Runner 群を構築・管理します。
type Manager struct {
	cfg           config.Config
	httpClient    httpkit.HTTPClient
	reader        remoteio.InputReader
	writer        remoteio.OutputWriter
	aiClient      gemini.GenerativeModel
	scriptPrompt  domain.ScriptPrompt
	imagePrompt   domain.ImagePrompt
	mangaComposer *generator.MangaComposer
	Runners       *Runners
}

// Runners は、構築済みの各 Runner を保持します。
type Runners struct {
	Design     runner.DesignRunner
	Script     runner.ScriptRunner
	PanelImage runner.PanelImageRunner
	PageImage  runner.PageImageRunner
	Publish    runner.PublishRunner
}

// New は、設定とキャラクター定義を基に新しい Manager を初期化します。
func New(ctx context.Context, args ManagerArgs) (*Manager, error) {
	// 1. 必須依存関係のバリデーション
	if err := validateArgs(args); err != nil {
		return nil, err
	}

	// 2. Config のデフォルト値補完
	cfg := args.Config
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = config.DefaultMaxConcurrency
	}
	if cfg.RateInterval <= 0 {
		cfg.RateInterval = config.DefaultRateInterval
	}
	if cfg.GeminiModel == "" {
		cfg.GeminiModel = config.DefaultGeminiModel
	}
	if cfg.ImageStandardModel == "" {
		cfg.ImageStandardModel = config.DefaultImageStandardModel
	}
	if cfg.ImageQualityModel == "" {
		cfg.ImageQualityModel = config.DefaultImageQualityModel
	}
	if cfg.StyleSuffix == "" {
		cfg.StyleSuffix = config.DefaultStyleSuffix
	}

	m := &Manager{
		cfg:          args.Config,
		httpClient:   args.HTTPClient,
		reader:       args.Reader,
		writer:       args.Writer,
		aiClient:     args.AIClient,
		scriptPrompt: args.ScriptPrompt,
		imagePrompt:  args.ImagePrompt,
	}

	var err error
	m.mangaComposer, err = m.buildMangaComposer(args.CharactersMap)
	if err != nil {
		return nil, err
	}

	m.Runners, err = m.buildAllRunners()
	if err != nil {
		return nil, err
	}

	return m, nil
}

// validateArgs 読みやすさのためにバリデーションを分離
func validateArgs(args ManagerArgs) error {
	if args.HTTPClient == nil {
		return fmt.Errorf("httpClient is required")
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
	if args.CharactersMap == nil {
		return fmt.Errorf("CharactersMap is required")
	}
	if args.ScriptPrompt == nil {
		return fmt.Errorf("ScriptPrompt is required")
	}
	if args.ImagePrompt == nil {
		return fmt.Errorf("ImagePrompt is required")
	}
	return nil
}
