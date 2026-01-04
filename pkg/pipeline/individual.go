package pipeline

import (
	"context"
	"fmt"

	"github.com/shouni/go-manga-kit/pkg/domain"

	//	imgkit "github.com/shouni/gemini-image-kit/pkg/adapters"
	imgdomain "github.com/shouni/gemini-image-kit/pkg/domain"
)

// IndividualPipeline は、全パネルを1枚ずつ順番に生成するパイプラインなのだ。
type IndividualPipeline struct {
	builder PromptBuilder
	adapter ImageAdapter
}

// NewIndividualPipeline は新しいパイプラインを生成するのだ。
func NewIndividualPipeline(b PromptBuilder, a ImageAdapter) *IndividualPipeline {
	return &IndividualPipeline{
		builder: b,
		adapter: a,
	}
}

// Execute は、パース済みの MangaResponse を受け取り、各パネルの画像を生成するのだ！
func (pl *IndividualPipeline) Execute(ctx context.Context, manga *domain.MangaResponse) ([]*imgdomain.ImageResponse, error) {
	if manga == nil {
		return nil, fmt.Errorf("individual_pipeline: manga data is nil なのだ")
	}

	results := make([]*imgdomain.ImageResponse, 0, len(manga.Pages))

	for _, page := range manga.Pages {
		// 1. プロンプトとシードの構築
		// [修正ポイント] BuildUnifiedPrompt は (string, int64) を返すようになったのだ。
		// 第2引数として speakerID（ここでは例として page.VisualAnchor 等のコンテキスト）を渡す想定なのだ。
		// ※ MangaPage 構造体に SpeakerID がある場合はそれを使うのだ。
		prompt, seedValue := pl.builder.BuildUnifiedPrompt(page)

		// TODO::core参照する [修正ポイント] Seed を *int64 として扱うのだ。
		var seedPtr *int64
		if seedValue != 0 {
			s := seedValue
			seedPtr = &s
		}

		// 2. リクエストの構築
		// [修正ポイント] ImageGenerationRequest.Seed は *int64 なのでそのまま渡せるのだ！
		req := imgdomain.ImageGenerationRequest{
			Prompt:       prompt,
			AspectRatio:  "16:9",
			Seed:         seedPtr,
			ReferenceURL: page.ReferenceURL,
		}

		// 3. 画像生成の実行
		resp, err := pl.adapter.GenerateMangaPanel(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("individual_pipeline: ページ %d の生成に失敗: %w", page.Page, err)
		}

		results = append(results, resp)
	}

	return results, nil
}
