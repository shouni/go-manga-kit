package workflow

import (
	"fmt"

	"github.com/shouni/go-prompt-kit/mdcast/builder"
	"github.com/shouni/go-web-exact/v2/pkg/extract"

	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-manga-kit/pkg/ports"
	"github.com/shouni/go-manga-kit/pkg/publisher"
	"github.com/shouni/go-manga-kit/pkg/runner"
)

// buildAllRunners は、ワークフローの実行に必要なすべてのランナーを構築して返します。
func (m *Manager) buildAllRunners() (*ports.Runners, error) {
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

	return &ports.Runners{
		Design:     dr,
		Script:     sr,
		PanelImage: panR,
		PageImage:  pagR,
		Publish:    pubR,
	}, nil
}

// buildScriptRunner は、台本生成を担当する Runner を作成します。
func (m *Manager) buildScriptRunner() (ports.ScriptRunner, error) {
	extractor, err := extract.NewExtractor(m.httpClient)
	if err != nil {
		return nil, fmt.Errorf("extractor の初期化に失敗しました: %w", err)
	}

	return runner.NewMangaScriptRunner(extractor, m.PromptDependencies.ScriptPrompt, m.aiClient, m.reader, m.cfg.GeminiModel), nil
}

// buildDesignRunner は、キャラクターデザインを担当する Runner を作成します。
func (m *Manager) buildDesignRunner() (ports.DesignRunner, error) {
	return runner.NewMangaDesignRunner(m.mangaComposer, m.writer, m.cfg.StyleSuffix), nil
}

// buildPanelImageRunner は、パネル画像生成を担当する Runner を作成します。
func (m *Manager) buildPanelImageRunner() (ports.PanelImageRunner, error) {
	panelsGen := generator.NewPanelGenerator(m.mangaComposer, m.PromptDependencies.ImagePrompt)

	return runner.NewMangaPanelRunner(panelsGen, m.writer), nil
}

// buildPageImageRunner は、Markdown からのページ画像一括生成を担当する Runner を作成します。
func (m *Manager) buildPageImageRunner() (ports.PageImageRunner, error) {
	pagesGen := generator.NewPageGenerator(m.mangaComposer, m.PromptDependencies.ImagePrompt, m.cfg.MaxPanelsPerPage)

	return runner.NewMangaPageRunner(pagesGen, m.reader, m.writer), nil
}

// buildPublishRunner は、成果物のパブリッシュを担当する Runner を作成します。
func (m *Manager) buildPublishRunner() (ports.PublishRunner, error) {
	b, err := builder.New()
	if err != nil {
		return nil, fmt.Errorf("mdcast builderの初期化に失敗: %w", err)
	}

	md2htmlRunner, err := b.BuildRunner()
	if err != nil {
		return nil, fmt.Errorf("MarkdownToHtmlRunnerの構築に失敗: %w", err)
	}

	pub := publisher.NewMangaPublisher(m.writer, md2htmlRunner)

	return runner.NewMangaPublisherRunner(pub), nil
}
