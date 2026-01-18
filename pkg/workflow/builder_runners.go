package workflow

import (
	"fmt"

	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-manga-kit/pkg/prompts"
	"github.com/shouni/go-manga-kit/pkg/publisher"
	"github.com/shouni/go-manga-kit/pkg/runner"

	"github.com/shouni/go-text-format/pkg/builder"
	"github.com/shouni/go-web-exact/v2/pkg/extract"
)

// BuildScriptRunner は、台本生成を担当する Runner を作成します。
func (b *Builder) BuildScriptRunner() (ScriptRunner, error) {
	extractor, err := extract.NewExtractor(b.httpClient)
	if err != nil {
		return nil, fmt.Errorf("extractor の初期化に失敗しました: %w", err)
	}

	pb, err := prompts.NewTextPromptBuilder()
	if err != nil {
		return nil, fmt.Errorf("prompt builder の作成に失敗しました: %w", err)
	}

	return runner.NewMangaScriptRunner(b.cfg, extractor, pb, b.aiClient, b.reader), nil
}

// BuildDesignRunner は、キャラクターデザインを担当する Runner を作成します。
func (b *Builder) BuildDesignRunner() (DesignRunner, error) {
	return runner.NewMangaDesignRunner(b.cfg, b.mangaComposer, b.writer), nil
}

// BuildPanelImageRunner は、パネル画像生成を担当する Runner を作成します。
func (b *Builder) BuildPanelImageRunner() (PanelImageRunner, error) {
	panelsGen := generator.NewPanelGenerator(b.mangaComposer)

	return runner.NewMangaPanelImageRunner(b.cfg, panelsGen, b.writer), nil
}

// BuildPageImageRunner は、Markdown からのページ画像一括生成を担当する Runner を作成します。
func (b *Builder) BuildPageImageRunner() (PageImageRunner, error) {
	pagesGen := generator.NewPageGenerator(b.mangaComposer)

	return runner.NewMangaPageRunner(b.cfg, pagesGen, b.reader, b.writer), nil
}

// BuildPublishRunner は、成果物のパブリッシュを担当する Runner を作成します。
func (b *Builder) BuildPublishRunner() (PublishRunner, error) {
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

	pub := publisher.NewMangaPublisher(b.mangaComposer.CharactersMap, b.writer, md2htmlRunner)
	return runner.NewDefaultPublisherRunner(b.cfg, pub), nil
}
