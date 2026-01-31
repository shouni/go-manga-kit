package workflow

import (
	"fmt"

	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-manga-kit/pkg/publisher"
	"github.com/shouni/go-manga-kit/pkg/runner"

	"github.com/shouni/go-text-format/pkg/builder"
	"github.com/shouni/go-web-exact/v2/pkg/extract"
)

// BuildScriptRunner は、台本生成を担当する Runner を作成します。
func (m *Manager) BuildScriptRunner() (ScriptRunner, error) {
	extractor, err := extract.NewExtractor(m.httpClient)
	if err != nil {
		return nil, fmt.Errorf("extractor の初期化に失敗しました: %w", err)
	}

	return runner.NewMangaScriptRunner(m.cfg, extractor, m.scriptPrompt, m.aiClient, m.reader), nil
}

// BuildDesignRunner は、キャラクターデザインを担当する Runner を作成します。
func (m *Manager) BuildDesignRunner() (DesignRunner, error) {
	return runner.NewMangaDesignRunner(m.cfg, m.mangaComposer, m.writer), nil
}

// BuildPanelImageRunner は、パネル画像生成を担当する Runner を作成します。
func (m *Manager) BuildPanelImageRunner() (PanelImageRunner, error) {
	panelsGen := generator.NewPanelGenerator(m.mangaComposer, m.imagePrompt)

	return runner.NewMangaPanelRunner(m.cfg, panelsGen, m.writer), nil
}

// BuildPageImageRunner は、Markdown からのページ画像一括生成を担当する Runner を作成します。
func (m *Manager) BuildPageImageRunner() (PageImageRunner, error) {
	pagesGen := generator.NewPageGenerator(m.mangaComposer, m.imagePrompt, m.cfg.MaxPanelsPerPage)

	return runner.NewMangaPageRunner(m.cfg, pagesGen, m.reader, m.writer), nil
}

// BuildPublishRunner は、成果物のパブリッシュを担当する Runner を作成します。
func (m *Manager) BuildPublishRunner() (PublishRunner, error) {
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

	return runner.NewMangaPublisherRunner(m.cfg, pub), nil
}
