package workflow

import (
	"fmt"

	"github.com/shouni/go-prompt-kit/md/builder"
	"github.com/shouni/go-web-exact/v2/extract"

	"github.com/shouni/go-manga-kit/layout"
	"github.com/shouni/go-manga-kit/ports"
	"github.com/shouni/go-manga-kit/publisher"
	"github.com/shouni/go-manga-kit/runner"
)

// buildAllRunners は、ワークフローの実行に必要なすべてのランナーを構築して返します。
func (m *manager) buildAllRunners() (*ports.Workflows, error) {
	dr, err := m.buildDesignRunner()
	if err != nil {
		return nil, fmt.Errorf("DesignRunner のビルドに失敗しました: %w", err)
	}
	sr, err := m.buildScriptRunner()
	if err != nil {
		return nil, fmt.Errorf("ScriptRunner のビルドに失敗しました: %w", err)
	}
	panR, err := m.buildPanelImageRunner()
	if err != nil {
		return nil, fmt.Errorf("PanelImageRunner のビルドに失敗しました: %w", err)
	}
	pagR, err := m.buildPageImageRunner()
	if err != nil {
		return nil, fmt.Errorf("PageImageRunner のビルドに失敗しました: %w", err)
	}
	pubR, err := m.buildPublishRunner()
	if err != nil {
		return nil, fmt.Errorf("PublishRunner のビルドに失敗しました: %w", err)
	}

	return &ports.Workflows{
		Design:     dr,
		Script:     sr,
		PanelImage: panR,
		PageImage:  pagR,
		Publish:    pubR,
	}, nil
}

// buildScriptRunner は、台本生成を担当する Runner を作成します。
func (m *manager) buildScriptRunner() (ports.ScriptRunner, error) {
	extractor, err := extract.NewExtractor(m.httpClient)
	if err != nil {
		return nil, fmt.Errorf("extractor の初期化に失敗しました: %w", err)
	}

	return runner.NewMangaScriptRunner(extractor, m.promptDeps.ScriptPrompt, m.aiClient, m.reader, m.cfg.GeminiModel), nil
}

// buildDesignRunner は、キャラクターデザインを担当する Runner を作成します。
func (m *manager) buildDesignRunner() (ports.DesignRunner, error) {
	llm := m.layoutManager.Quality
	return runner.NewMangaDesignRunner(
		llm.mangaComposer,
		llm.imageGenerator,
		m.writer,
		llm.model, // LLM ユニットに保持されたモデル名を使用
		m.cfg.StyleSuffix,
	), nil
}

// buildPanelImageRunner は、パネル画像生成を担当する Runner を作成します。
func (m *manager) buildPanelImageRunner() (ports.PanelImageRunner, error) {
	llm := m.layoutManager.Standard
	panelsGen := layout.NewPanelGenerator(
		llm.mangaComposer,
		llm.imageGenerator,
		m.promptDeps.ImagePrompt,
		llm.model,
		layout.WithPanelMaxConcurrency(m.cfg.MaxConcurrency),
		layout.WithPanelRateInterval(m.cfg.RateInterval),
	)

	return runner.NewMangaPanelRunner(panelsGen, m.writer), nil
}

// buildPageImageRunner は、Markdown からのページ画像一括生成を担当する Runner を作成します。
func (m *manager) buildPageImageRunner() (ports.PageImageRunner, error) {
	llm := m.layoutManager.Quality
	pagesGen := layout.NewPageGenerator(
		llm.mangaComposer,
		llm.imageGenerator,
		m.promptDeps.ImagePrompt,
		llm.model,
		layout.WithPageRateInterval(m.cfg.RateInterval),
		layout.WithMaxPanelsPerPage(m.cfg.MaxPanelsPerPage),
	)

	return runner.NewMangaPageRunner(pagesGen, m.writer), nil
}

// buildPublishRunner は、成果物のパブリッシュを担当する Runner を作成します。
func (m *manager) buildPublishRunner() (ports.PublishRunner, error) {
	b, err := builder.New()
	if err != nil {
		return nil, fmt.Errorf("MarkdownBuilderの初期化に失敗: %w", err)
	}

	md2htmlRunner, err := b.BuildRunner()
	if err != nil {
		return nil, fmt.Errorf("MarkdownToHtmlRunnerの構築に失敗: %w", err)
	}

	pub := publisher.NewMangaPublisher(m.writer, md2htmlRunner)

	return runner.NewMangaPublisherRunner(pub), nil
}
