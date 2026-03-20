package workflow

import (
	"fmt"

	"github.com/patrickmn/go-cache"
	imagekit "github.com/shouni/gemini-image-kit/pkg/generator"
	"golang.org/x/time/rate"

	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-manga-kit/pkg/ports"
)

// buildMangaComposer 提供された構成と依存関係を使用して MangaComposer インスタンスを初期化し、返します。
func (m *Manager) buildMangaComposer(
	chars ports.CharactersMap,
) (*generator.MangaComposer, error) {
	// 画像生成エンジンの初期化
	core, err := imagekit.NewGeminiImageCore(
		m.aiClient,
		m.reader,
		m.httpClient,
		cache.New(defaultCacheExpiration, cacheCleanupInterval),
		defaultTTL,
		false,
	)
	if err != nil {
		return nil, fmt.Errorf("画像生成エンジンの初期化に失敗しました: %w", err)
	}

	imageGenerator, err := imagekit.NewGeminiGenerator(
		m.cfg.ImageStandardModel,
		m.cfg.ImageQualityModel,
		core,
	)
	if err != nil {
		return nil, fmt.Errorf("GeminiGeneratorの初期化に失敗しました: %w", err)
	}

	composer, err := generator.NewMangaComposer(
		core,
		imageGenerator,
		chars,
		rate.NewLimiter(rate.Every(m.cfg.RateInterval), defaultRateBurst),
		m.cfg.MaxConcurrency,
	)
	if err != nil {
		return nil, fmt.Errorf("MangaComposerの初期化に失敗しました: %w", err)
	}

	return composer, nil
}
