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

const (
	defaultCacheExpiration = 5 * time.Minute
	cacheCleanupInterval   = 15 * time.Minute
	defaultTTL             = 5 * time.Minute
)

// Builder はワークフローの各工程を担う Runner 群を構築・管理するのだ。
type Builder struct {
	cfg        config.Config
	chars      domain.CharactersMap
	httpClient httpkit.ClientInterface
	aiClient   gemini.GenerativeModel
	reader     remoteio.InputReader
	writer     remoteio.OutputWriter
	imgGen     imageKit.ImageGenerator
	mangaGen   generator.MangaGenerator
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
	imgGen, err := initializeImageGenerator(httpClient, aiClient, cfg.ImageModel)
	if err != nil {
		return nil, fmt.Errorf("failed to load characters: %w", err)
	}

	mangaGen := generator.MangaGenerator{
		ImgGen:     imgGen,
		Characters: chars,
	}

	return &Builder{
		cfg:        cfg,
		chars:      chars,
		httpClient: httpClient,
		aiClient:   aiClient,
		reader:     reader,
		writer:     writer,
		imgGen:     imgGen,
		mangaGen:   mangaGen,
	}, nil
}

// BuildScriptRunner は台本生成を担当する Runner を作成するのだ。
func (b *Builder) BuildScriptRunner() (ScriptRunner, error) {
	extractor, err := extract.NewExtractor(b.httpClient)
	if err != nil {
		return nil, fmt.Errorf("エクストラクタの初期化に失敗しました: %w", err)
	}

	pb, err := prompts.NewTextPromptBuilder()
	if err != nil {
		return nil, fmt.Errorf("プロンプトビルダーの作成に失敗しました: %w", err)
	}

	return runner.NewMangaScriptRunner(b.cfg, extractor, pb, b.aiClient, b.reader), nil
}

// BuildDesignRunner はキャラクターデザインを担当する Runner を作成するのだ。
func (b *Builder) BuildDesignRunner() DesignRunner {
	return runner.NewMangaDesignRunner(b.cfg, b.mangaGen, b.writer)
}

// BuildPanelImageRunner はパネル並列生成を担当する Runner を作成するのだ。
func (b *Builder) BuildPanelImageRunner() PanelImageRunner {
	return runner.NewMangaPanelImageRunner(b.cfg, b.mangaGen, b.cfg.StyleSuffix, b.cfg.RateInterval)
}

// BuildPageImageRunner は Markdown からの一括生成を担当する Runner を作成するのだ。
func (b *Builder) BuildPageImageRunner() PageImageRunner {
	return runner.NewMangaPageRunner(b.cfg, b.mangaGen, b.cfg.StyleSuffix)
}

// BuildPublishRunner は成果物のパブリッシュを担当する Runner を作成するのだ。
func (b *Builder) BuildPublishRunner() (PublishRunner, error) {
	htmlCfg := builder.BuilderConfig{
		EnableHardWraps: true,
		Mode:            "webtoon",
	}
	md2htmlBuilder, err := builder.NewBuilder(htmlCfg)
	if err != nil {
		return nil, fmt.Errorf("md2htmlBuilderの初期化に失敗しました: %w", err)
	}
	md2htmlRunner, err := md2htmlBuilder.BuildRunner()
	if err != nil {
		return nil, fmt.Errorf("md2htmlrunnerの初期化に失敗しました: %w", err)
	}

	pub := publisher.NewMangaPublisher(b.writer, md2htmlRunner)

	return runner.NewDefaultPublisherRunner(b.cfg, pub), nil
}

// initializeImageGenerator は ImageGeneratorを初期化します。
func initializeImageGenerator(httpClient httpkit.ClientInterface, aiClient gemini.GenerativeModel, model string) (imageKit.ImageGenerator, error) {
	imgCache := cache.New(defaultCacheExpiration, cacheCleanupInterval)
	// 画像処理コアを生成
	core, err := imageKit.NewGeminiImageCore(
		httpClient,
		imgCache,
		defaultTTL,
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
