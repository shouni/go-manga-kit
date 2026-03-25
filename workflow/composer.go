package workflow

import (
	"fmt"

	"github.com/patrickmn/go-cache"
	"github.com/shouni/gemini-image-kit/generator"
	imagePorts "github.com/shouni/gemini-image-kit/ports"
	"golang.org/x/time/rate"

	"github.com/shouni/go-manga-kit/layout"
	"github.com/shouni/go-manga-kit/ports"
)

// buildComposerAndGenerator は提供された構成と依存関係を使用して MangaComposer と ImageGenerator インスタンスを初期化し、返します。
func (m *manager) buildComposerAndGenerator(
	chars ports.CharactersMap,
) (*layout.MangaComposer, imagePorts.ImageGenerator, error) {
	// 画像生成エンジンの初期化
	core, err := generator.NewGeminiImageCore(
		m.aiClient,
		m.reader,
		m.httpClient,
		cache.New(defaultCacheExpiration, cacheCleanupInterval),
		defaultTTL,
		false,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("画像生成エンジンの初期化に失敗しました: %w", err)
	}

	imageGenerator, err := generator.NewGeminiGenerator(
		m.cfg.ImageStandardModel,
		m.cfg.ImageQualityModel,
		core,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("GeminiGeneratorの初期化に失敗しました: %w", err)
	}

	composer, err := layout.NewMangaComposer(
		core,
		imageGenerator,
		chars,
		rate.NewLimiter(rate.Every(m.cfg.RateInterval), defaultRateBurst),
		m.cfg.MaxConcurrency,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("MangaComposerの初期化に失敗しました: %w", err)
	}

	return composer, imageGenerator, nil
}
