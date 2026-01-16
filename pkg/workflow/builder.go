package workflow

import (
	"fmt"

	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-manga-kit/pkg/prompts"

	"github.com/patrickmn/go-cache"
	imageKit "github.com/shouni/gemini-image-kit/pkg/generator"
	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

// Builder は、ワークフローの各工程を担う Runner 群を構築・管理します。
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

// NewBuilder は、設定とキャラクター定義を基に新しい Builder を初期化します。
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

	// 1. キャラクターデータの解析
	chars, err := domain.GetCharacters(charData)
	if err != nil {
		return nil, fmt.Errorf("キャラクターデータの解析に失敗しました: %w", err)
	}

	// 2. 画像生成エンジンの初期化
	imgGen, err := initializeImageGenerator(reader, httpClient, aiClient, cfg.ImageModel)
	if err != nil {
		return nil, fmt.Errorf("画像生成エンジンの初期化に失敗しました: %w", err)
	}

	pb := prompts.NewImagePromptBuilder(chars, cfg.StyleSuffix)
	mangaGen := generator.MangaGenerator{
		ImgGen:        imgGen,
		PromptBuilder: pb,
		Characters:    chars,
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

// initializeImageGenerator は、画像キャッシュを含む ImageGenerator を初期化します。
func initializeImageGenerator(reader remoteio.InputReader, httpClient httpkit.ClientInterface, aiClient gemini.GenerativeModel, model string) (imageKit.ImageGenerator, error) {
	imgCache := cache.New(defaultCacheExpiration, cacheCleanupInterval)
	core, err := imageKit.NewGeminiImageCore(
		reader,
		httpClient,
		imgCache,
		defaultTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("GeminiImageCore の初期化に失敗しました: %w", err)
	}

	imgGen, err := imageKit.NewGeminiGenerator(
		core,
		aiClient,
		model,
	)
	if err != nil {
		return nil, fmt.Errorf("GeminiGenerator の初期化に失敗しました: %w", err)
	}

	return imgGen, nil
}
