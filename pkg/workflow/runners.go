package workflow

import (
	"fmt"

	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-manga-kit/pkg/publisher"
	"github.com/shouni/go-manga-kit/pkg/runner"

	"github.com/shouni/go-text-format/pkg/builder"
	"github.com/shouni/go-web-exact/v2/pkg/extract"
)

// buildAllRunners は、ワークフローの実行に必要なすべてのランナーを構築して返します。
func (m *Manager) buildAllRunners() (*Runners, error) {
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

	return &Runners{
		Design:     dr,
		Script:     sr,
		PanelImage: panR,
		PageImage:  pagR,
		Publish:    pubR,
	}, nil
}

// buildScriptRunner は、台本生成を担当する Runner を作成します。
func (m *Manager) buildScriptRunner() (runner.ScriptRunner, error) {
	extractor, err := extract.NewExtractor(m.httpClient)
	if err != nil {
		return nil, fmt.Errorf("extractor の初期化に失敗しました: %w", err)
	}

	return runner.NewMangaScriptRunner(extractor, m.scriptPrompt, m.aiClient, m.reader, m.cfg.GeminiModel), nil
}

// buildDesignRunner は、キャラクターデザインを担当する Runner を作成します。
func (m *Manager) buildDesignRunner() (runner.DesignRunner, error) {
	return runner.NewMangaDesignRunner(m.mangaComposer, m.writer, m.cfg.StyleSuffix), nil
}

// buildPanelImageRunner は、パネル画像生成を担当する Runner を作成します。
func (m *Manager) buildPanelImageRunner() (runner.PanelImageRunner, error) {
	panelsGen := generator.NewPanelGenerator(m.mangaComposer, m.imagePrompt)

	return runner.NewMangaPanelRunner(panelsGen, m.writer), nil
}

// buildPageImageRunner は、Markdown からのページ画像一括生成を担当する Runner を作成します。
func (m *Manager) buildPageImageRunner() (runner.PageImageRunner, error) {
	pagesGen := generator.NewPageGenerator(m.mangaComposer, m.imagePrompt, m.cfg.MaxPanelsPerPage)

	return runner.NewMangaPageRunner(pagesGen, m.reader, m.writer), nil
}

// buildPublishRunner は、成果物のパブリッシュを担当する Runner を作成します。
func (m *Manager) buildPublishRunner() (runner.PublishRunner, error) {
	htmlCfg := builder.BuilderConfig{
		EnableHardWraps: true,
		Mode:            "webtoon",
	}
	md2htmlBuilder, err := builder.NewBuilder(htmlCfg)
	if err != nil {
		return nil, fmt.Errorf("md2htmlBuilder の初期化に失敗しました: %w", err)
	}
	md2htmlRunner, err := md2htmlBuilder.BuildRunner()
	if err != nil {
		return nil, fmt.Errorf("md2htmlRunner の初期化に失敗しました: %w", err)
	}

	pub := publisher.NewMangaPublisher(m.writer, md2htmlRunner)

	return runner.NewMangaPublisherRunner(pub), nil
}
