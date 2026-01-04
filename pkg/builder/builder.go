package builder

import (
	"context"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	imageKit "github.com/shouni/gemini-image-kit/pkg/adapters"
	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-manga-kit/pkg/adapters" // ここにPipelineやインターフェースがある
	"google.golang.org/genai"
)

// InitializeAIClient は Gemini クライアントを初期化するのだ。
func InitializeAIClient(ctx context.Context, apiKey string) (gemini.GenerativeModel, error) {
	const defaultGeminiTemperature = float32(0.2)
	clientConfig := gemini.Config{
		APIKey:      apiKey,
		Temperature: genai.Ptr(defaultGeminiTemperature),
	}
	aiClient, err := gemini.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("AIクライアントの初期化に失敗なのだ: %w", err)
	}
	return aiClient, nil
}

// InitializeImageCore は共通の画像処理コアを生成するのだ。キャッシュ管理もここなのだ！
func InitializeImageCore(clientInterface httpkit.ClientInterface) imageKit.ImageGeneratorCore {
	imgCache := cache.New(30*time.Minute, 1*time.Hour)
	cacheTTL := 1 * time.Hour

	return imageKit.NewGeminiImageCore(
		clientInterface,
		imgCache,
		cacheTTL,
	)
}

// InitializeImageAdapter は、個別パネル生成（GroupPipeline用）のアダプターを初期化するのだ。
func InitializeImageAdapter(core imageKit.ImageGeneratorCore, aiClient gemini.GenerativeModel, imageModel, promptSuffix string) (adapters.ImageAdapter, error) {
	return imageKit.NewGeminiImageAdapter(
		core,
		aiClient,
		imageModel,
		promptSuffix,
	)
}

// InitializeMangaPageAdapter は、1枚のページ生成（PagePipeline用）のアダプターを初期化するのだ。
func InitializeMangaPageAdapter(core imageKit.ImageGeneratorCore, aiClient gemini.GenerativeModel, imageModel string) adapters.MangaPageAdapter {
	return imageKit.NewGeminiMangaPageAdapter(
		core,
		aiClient,
		imageModel,
	)
}
