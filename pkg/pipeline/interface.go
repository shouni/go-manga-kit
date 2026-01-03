package pipeline

import (
	"context"

	"github.com/shouni/go-manga-kit/pkg/domain"

	imgkit "github.com/shouni/gemini-image-kit/pkg/domain"
)

type Parser interface {
	Parse(input string) (*domain.MangaResponse, error)
}

type PromptBuilder interface {
	BuildUnifiedPrompt(page domain.MangaPage) (string, int64)
	BuildFullPagePrompt(title string, panels []string, chars []domain.Character) string
}

type ImageAdapter interface {
	GenerateMangaPanel(ctx context.Context, req imgkit.ImageGenerationRequest) (*imgkit.ImageResponse, error)
}

type MangaPageAdapter interface {
	GenerateMangaPage(ctx context.Context, req imgkit.ImagePageRequest) (*imgkit.ImageResponse, error)
}
