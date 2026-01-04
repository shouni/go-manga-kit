package pipeline

import (
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/shouni/gemini-image-kit/pkg/generator"
	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
)

// InitializeImageGenerator は ImageGeneratorを初期化します。
func InitializeImageGenerator(httpClient httpkit.ClientInterface, aiClient gemini.GenerativeModel, model string) (generator.ImageGenerator, error) {
	imgCache := cache.New(30*time.Minute, 1*time.Hour)
	cacheTTL := 1 * time.Hour

	// 画像処理コアを生成
	core, err := generator.NewGeminiImageCore(
		httpClient,
		imgCache,
		cacheTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("GeminiImageCoreの初期化に失敗したのだ: %w", err)
	}

	imgGen, err := generator.NewGeminiGenerator(
		core,
		aiClient,
		model,
	)
	if err != nil {
		return nil, fmt.Errorf("GeminiGeneratorの初期化に失敗したのだ: %w", err)
	}

	return imgGen, nil
}
