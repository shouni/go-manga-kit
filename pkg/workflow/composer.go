package workflow

import (
	"fmt"

	"github.com/patrickmn/go-cache"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/generator"
	"golang.org/x/time/rate"

	imagekit "github.com/shouni/gemini-image-kit/pkg/generator"

	"github.com/shouni/go-gemini-client/pkg/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

const defaultRateBurst = 2

// buildMangaComposer 提供された構成と依存関係を使用して MangaComposer インスタンスを初期化し、返します。
func (m *Manager) buildMangaComposer(
	chars domain.CharactersMap,
) (*generator.MangaComposer, error) {
	// 画像生成エンジンの初期化
	core, err := initializeCore(m.reader, m.httpClient, m.aiClient)
	if err != nil {
		return nil, fmt.Errorf("画像生成エンジンの初期化に失敗しました: %w", err)
	}
	assetManager := initializeAssetManager(core)
	imageGenerator, err := initializeImageGenerator(m.cfg.ImageModel, core)
	if err != nil {
		return nil, fmt.Errorf("ImageGeneratorの初期化に失敗しました: %w", err)
	}

	return generator.NewMangaComposer(
		assetManager,
		imageGenerator,
		chars,
		rate.NewLimiter(rate.Every(m.cfg.RateInterval), defaultRateBurst),
	), nil
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
