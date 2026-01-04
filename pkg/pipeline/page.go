package pipeline

import (
	"context"
	"fmt"

	imgdomain "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/domain"
)

// PagePipeline は1枚の統合ページを錬成するのだ
type PagePipeline struct {
	parser  Parser
	builder PromptBuilder
	adapter MangaPageAdapter
}

func NewPagePipeline(p Parser, b PromptBuilder, a MangaPageAdapter) *PagePipeline {
	return &PagePipeline{
		parser:  p,
		builder: b,
		adapter: a,
	}
}

func (pl *PagePipeline) Execute(ctx context.Context, markdown string, characters []domain.Character) (*imgdomain.ImageResponse, error) {
	manga, err := pl.parser.Parse(markdown)
	if err != nil {
		return nil, fmt.Errorf("page_pipeline: 解析失敗: %w", err)
	}

	panelPrompts := make([]string, 0, len(manga.Pages))
	refURLs := make([]string, 0)
	var baseSeed *int64

	for i, page := range manga.Pages {
		prompt, seed := pl.builder.BuildUnifiedPrompt(page)
		panelPrompts = append(panelPrompts, prompt)
		if i == 0 && seed != 0 {
			baseSeed = &seed
		}
		if page.ReferenceURL != "" {
			refURLs = append(refURLs, page.ReferenceURL)
		}
	}

	fullPrompt := pl.builder.BuildFullPagePrompt(manga.Title, panelPrompts, characters)

	req := imgdomain.ImagePageRequest{
		Prompt:         fullPrompt,
		NegativePrompt: "deformed, text bubbles",
		AspectRatio:    "3:4",
		Seed:           baseSeed,
		ReferenceURLs:  refURLs,
	}

	return pl.adapter.GenerateMangaPage(ctx, req)
}
