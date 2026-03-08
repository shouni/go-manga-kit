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
	if args.HTTPClient == nil {
		return nil, fmt.Errorf("httpClient は必須です")
	}
	if args.Reader == nil {
		return nil, fmt.Errorf("InputReader は必須です")
	}
	if args.Writer == nil {
		return nil, fmt.Errorf("OutputWriter は必須です")
	}
	if args.AIClient == nil {
		return nil, fmt.Errorf("AIClient は必須です")
	}
	if args.CharactersMap == nil {
		return nil, fmt.Errorf("CharactersMap は必須です")
	}
	if args.ScriptPrompt == nil {
		return nil, fmt.Errorf("ScriptPrompt は必須です")
	}
	if args.ImagePrompt == nil {
		return nil, fmt.Errorf("ImagePrompt は必須です")
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
