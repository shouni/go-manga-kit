package adapters

import (
	"context"
	"fmt"

	"github.com/shouni/go-manga-kit/pkg/domain"

	// imgkit "github.com/shouni/gemini-image-kit/pkg/adapters"
	imgdomain "github.com/shouni/gemini-image-kit/pkg/domain"
)

// PanelGroupPipeline は、複数のパネルを「群」として順次生成を管理するパイプラインなのだ。
type PanelGroupPipeline struct {
	builder PromptBuilder
	adapter ImageAdapter
}

// NewPanelGroupPipeline は新しいパネル群生成パイプラインを作成するのだ。
func NewPanelGroupPipeline(b PromptBuilder, a ImageAdapter) *PanelGroupPipeline {
	return &PanelGroupPipeline{
		builder: b,
		adapter: a,
	}
}

// Execute は、パース済みの MangaResponse を受け取り、各パネルの画像を生成するのだ！
func (pl *PanelGroupPipeline) Execute(ctx context.Context, manga *domain.MangaResponse) ([]*imgdomain.ImageResponse, error) {
	if manga == nil {
		return nil, fmt.Errorf("panel_group_pipeline: manga data is nil なのだ")
	}

	results := make([]*imgdomain.ImageResponse, 0, len(manga.Pages))

	for _, page := range manga.Pages {
		// 1. プロンプトとシードの構築
		// BuildUnifiedPrompt は (string, int64) を返す想定なのだ。
		prompt, seedValue := pl.builder.BuildUnifiedPrompt(page)

		// シード値の処理 (0の場合はnilとして扱うなどのロジック)
		var seedPtr *int64
		if seedValue != 0 {
			s := seedValue
			seedPtr = &s
		}

		// 2. リクエストの構築
		req := imgdomain.ImageGenerationRequest{
			Prompt:       prompt,
			AspectRatio:  "16:9",
			Seed:         seedPtr, // ポインタとして渡すのだ！
			ReferenceURL: page.ReferenceURL,
		}

		// 3. 画像生成の実行
		resp, err := pl.adapter.GenerateMangaPanel(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("panel_group_pipeline: ページ %d の生成に失敗: %w", page.Page, err)
		}

		results = append(results, resp)
	}

	return results, nil
}
