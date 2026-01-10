package workflow

import (
	"fmt"
	"time"

	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-manga-kit/pkg/prompts"
	"github.com/shouni/go-manga-kit/pkg/publisher"
	"github.com/shouni/go-manga-kit/pkg/runner"

	"github.com/patrickmn/go-cache"
	imageKit "github.com/shouni/gemini-image-kit/pkg/generator"
	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"github.com/shouni/go-text-format/pkg/builder"
	"github.com/shouni/go-web-exact/v2/pkg/extract"
)

// Builder はワークフローの各工程を担う Runner 群を構築・管理するのだ。
type Builder struct {
	cfg        config.Config
	httpClient httpkit.ClientInterface
	aiClient   gemini.GenerativeModel
	imgGen     imageKit.ImageGenerator
	reader     remoteio.InputReader
	writer     remoteio.OutputWriter
	chars      domain.CharactersMap
}

// NewBuilder は Config と必要なキャラクター定義を基に新しい Builder を作成するのだ。
func NewBuilder(cfg config.Config, httpClient httpkit.ClientInterface, aiClient gemini.GenerativeModel, reader remoteio.InputReader, writer remoteio.OutputWriter, charData []byte) (*Builder, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("httpClient は必須です")
	}
	if aiClient == nil {
		return nil, fmt.Errorf("aiClient は必須です")
	}
	if reader == nil {
		return nil, fmt.Errorf("reader は必須です")
	}
	if writer == nil {
		return nil, fmt.Errorf("writer は必須です")
	}

	// 1. キャラクターデータのパース
	chars, err := domain.GetCharacters(charData)
	if err != nil {
		return nil, fmt.Errorf("failed to load characters: %w", err)
	}
	imgGen, err := initializeImageGenerator(httpClient, aiClient, cfg.GeminiModel)
	if err != nil {
		return nil, fmt.Errorf("failed to load characters: %w", err)
	}

	return &Builder{
		cfg:        cfg,
		httpClient: httpClient,
		aiClient:   aiClient,
		imgGen:     imgGen,
		reader:     reader,
		writer:     writer,
		chars:      chars,
	}, nil
}

// BuildScriptRunner は台本生成を担当する Runner を作成するのだ。
func (b *Builder) BuildScriptRunner() ScriptRunner {
	extractor, _ := extract.NewExtractor(b.httpClient)
	// prompts パッケージの Builder を使用
	pb, _ := prompts.NewTextPromptBuilder()

	return runner.NewMangaScriptRunner(b.cfg, extractor, pb, b.aiClient, b.reader)
}

// BuildDesignRunner はキャラクターデザインを担当する Runner を作成するのだ。
func (b *Builder) BuildDesignRunner() DesignRunner {
	mangaGen := generator.MangaGenerator{
		ImgGen:     b.imgGen,
		Characters: b.chars,
	}
	return runner.NewMangaDesignRunner(b.cfg, mangaGen, b.writer)
}

// BuildPanelImageRunner はパネル並列生成を担当する Runner を作成するのだ。
func (b *Builder) BuildPanelImageRunner() PanelImageRunner {
	mangaGen := generator.MangaGenerator{
		ImgGen:     b.imgGen,
		Characters: b.chars,
	}

	return runner.NewMangaPanelImageRunner(b.cfg, mangaGen, b.cfg.StyleSuffix, b.cfg.RateInterval)
}

// BuildPageImageRunner は Markdown からの一括生成を担当する Runner を作成するのだ。
func (b *Builder) BuildPageImageRunner() PageImageRunner {
	mangaGen := generator.MangaGenerator{
		ImgGen:     b.imgGen,
		Characters: b.chars,
	}
	return runner.NewMangaPageRunner(b.cfg, mangaGen, b.cfg.StyleSuffix)
}

// BuildPublishRunner は成果物のパブリッシュを担当する Runner を作成するのだ。
func (b *Builder) BuildPublishRunner() PublishRunner {
	config := builder.BuilderConfig{
		EnableHardWraps: true,
		Mode:            "webtoon",
	}
	md2htmlBuilder, _ := builder.NewBuilder(config)
	md2htmlRunner, _ := md2htmlBuilder.BuildRunner()

	pub := publisher.NewMangaPublisher(b.writer, md2htmlRunner)
	return runner.NewDefaultPublisherRunner(b.cfg, pub)
}

// initializeImageGenerator は ImageGeneratorを初期化します。
func initializeImageGenerator(httpClient httpkit.ClientInterface, aiClient gemini.GenerativeModel, model string) (imageKit.ImageGenerator, error) {
	imgCache := cache.New(30*time.Minute, 1*time.Hour)
	cacheTTL := 1 * time.Hour

	// 画像処理コアを生成
	core, err := imageKit.NewGeminiImageCore(
		httpClient,
		imgCache,
		cacheTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("GeminiImageCoreの初期化に失敗しました: %w", err)
	}

	imgGen, err := imageKit.NewGeminiGenerator(
		core,
		aiClient,
		model,
	)
	if err != nil {
		return nil, fmt.Errorf("GeminiGeneratorの初期化に失敗しました: %w", err)
	}

	return imgGen, nil
}
