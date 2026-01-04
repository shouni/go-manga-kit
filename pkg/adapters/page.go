package adapters

import (
	"context"
	"fmt"

	imgdomain "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/domain"
)

// PagePipeline は、複数のパネルを統合して「1枚の漫画ページ」を生成するパイプラインなのだ。
type PagePipeline struct {
	parser  Parser
	builder PromptBuilder
	adapter MangaPageAdapter
}

// NewPagePipeline は新しいページ生成パイプラインを生成するのだ。
func NewPagePipeline(p Parser, b PromptBuilder, a MangaPageAdapter) *PagePipeline {
	return &PagePipeline{
		parser:  p,
		builder: b,
		adapter: a,
	}
}

// Execute は、Markdownから全パネル情報を解析し、1枚の統合されたページ画像を生成するのだ！
func (pl *PagePipeline) Execute(ctx context.Context, markdown string, characters []domain.Character) (*imgdomain.ImageResponse, error) {
	// 1. Markdownの解析
	manga, err := pl.parser.Parse(markdown)
	if err != nil {
		return nil, fmt.Errorf("page_pipeline: 解析失敗なのだ: %w", err)
	}

	panelPrompts := make([]string, 0, len(manga.Pages))
	refURLs := make([]string, 0)
	var baseSeed *int64

	// 2. 各パネル（群）から情報を集約するのだ
	for i, page := range manga.Pages {
		prompt, seed := pl.builder.BuildUnifiedPrompt(page)
		panelPrompts = append(panelPrompts, prompt)

		// 最初のパネルのシードをページ全体のベースシードとして採用するのだ
		if i == 0 && seed != 0 {
			s := seed
			baseSeed = &s
		}
		if page.ReferenceURL != "" {
			refURLs = append(refURLs, page.ReferenceURL)
		}
	}

	// 3. 全パネルを統合したフルページプロンプトの構築
	fullPrompt := pl.builder.BuildFullPagePrompt(manga.Title, panelPrompts, characters)

	// 4. ページ生成リクエストの構築
	req := imgdomain.ImagePageRequest{
		Prompt:         fullPrompt,
		NegativePrompt: "deformed, text bubbles, extra fingers", // ネガティブプロンプトを少し強化したのだ
		AspectRatio:    "3:4",
		Seed:           baseSeed,
		ReferenceURLs:  refURLs,
	}

	// 5. 画像生成（ページ単位）の実行
	return pl.adapter.GenerateMangaPage(ctx, req)
}
