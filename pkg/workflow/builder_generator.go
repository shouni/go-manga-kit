package workflow

import (
	"fmt"

	"github.com/patrickmn/go-cache"
	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-manga-kit/pkg/prompts"
	"golang.org/x/time/rate"

	imagekit "github.com/shouni/gemini-image-kit/pkg/generator"

	"github.com/shouni/go-gemini-client/pkg/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

// buildMangaComposer 提供された構成と依存関係を使用して MangaComposer インスタンスを初期化し、返します。
func buildMangaComposer(
	cfg config.Config,
	httpClient httpkit.ClientInterface,
	aiClient gemini.GenerativeModel,
	reader remoteio.InputReader,
	charData []byte) (*generator.MangaComposer, error) {

	// 1. キャラクターデータの解析
	chars, err := domain.GetCharacters(charData)
	if err != nil {
		return nil, fmt.Errorf("キャラクターデータの解析に失敗しました: %w", err)
	}

	// 2. 画像生成エンジンの初期化
	core, err := initializeCore(reader, httpClient, aiClient)
	if err != nil {
		return nil, fmt.Errorf("画像生成エンジンの初期化に失敗しました: %w", err)
	}
	assetManager := initializeAssetManager(core)
	imageGenerator, err := initializeImageGenerator(cfg.ImageModel, core)
	if err != nil {
		return nil, fmt.Errorf("画像生成エンジンの初期化に失敗しました: %w", err)
	}
	pb := prompts.NewImagePromptBuilder(chars, cfg.StyleSuffix)

	return &generator.MangaComposer{
		AssetManager:   assetManager,
		ImageGenerator: imageGenerator,
		PromptBuilder:  pb,
		CharactersMap:  chars,
		RateLimiter:    rate.NewLimiter(rate.Every(cfg.RateInterval), 2),
	}, nil
}

// initializeAssetManager 提供された GeminiImageCore を使用して AssetManager インスタンスを初期化し、返します。
func initializeAssetManager(core *imagekit.GeminiImageCore) imagekit.AssetManager {
	return core
}

// initializeImageGenerator は、画像キャッシュを含む ImageGenerator を初期化します。
func initializeImageGenerator(model string, core *imagekit.GeminiImageCore) (imagekit.ImageGenerator, error) {
	return imagekit.NewGeminiGenerator(
		model,
		core,
	)
}

// initializeCore 提供された依存関係で構成された GeminiImageCore インスタンスを初期化して返します。
func initializeCore(reader remoteio.InputReader, httpClient httpkit.ClientInterface, aiClient gemini.GenerativeModel) (*imagekit.GeminiImageCore, error) {
	imgCache := cache.New(defaultCacheExpiration, cacheCleanupInterval)
	core, err := imagekit.NewGeminiImageCore(
		aiClient,
		reader,
		httpClient,
		imgCache,
		defaultTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("GeminiImageCore の初期化に失敗しました: %w", err)
	}

	return core, nil
}
