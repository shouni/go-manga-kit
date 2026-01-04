package adapters

import (
	"context"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
)

// ImageAdapter は個別パネル（1枚）の生成を担うのだ
type ImageAdapter interface {
	GenerateMangaPanel(ctx context.Context, req imagedom.ImageGenerationRequest) (*imagedom.ImageResponse, error)
}

// MangaPageAdapter は複数パネルが統合されたページ生成を担うのだ
type MangaPageAdapter interface {
	GenerateMangaPage(ctx context.Context, req imagedom.ImagePageRequest) (*imagedom.ImageResponse, error)
}

// PromptBuilder はプロンプト構築のロジックを抽象化するのだ
type PromptBuilder interface {
	BuildUnifiedPrompt(page interface{}) (string, int64)
	BuildFullPagePrompt(title string, panels []string, characters interface{}) string
}
