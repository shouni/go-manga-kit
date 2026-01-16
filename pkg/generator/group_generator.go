package generator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/prompts"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// GroupGenerator は、キャラクターの一貫性を保ちながら並列で複数パネルを生成します。
type GroupGenerator struct {
	mangaGenerator MangaGenerator
	limiter        *rate.Limiter
}

// NewGroupGenerator は GroupGenerator の新しいインスタンスを初期化します。
func NewGroupGenerator(mangaGenerator MangaGenerator, interval time.Duration) *GroupGenerator {
	return &GroupGenerator{
		mangaGenerator: mangaGenerator,
		limiter:        rate.NewLimiter(rate.Every(interval), 2),
	}
}

// Execute は、並列処理を用いてパネル群を生成します。
func (gg *GroupGenerator) Execute(ctx context.Context, panels []domain.Panel) ([]*imagedom.ImageResponse, error) {
	// プロンプトビルダーの取得
	pb := gg.mangaGenerator.PromptBuilder
	images := make([]*imagedom.ImageResponse, len(panels))
	eg, egCtx := errgroup.WithContext(ctx)

	for i, panel := range panels {
		i, panel := i, panel
		eg.Go(func() error {
			if err := gg.limiter.Wait(egCtx); err != nil {
				return err
			}

			// キャラクター解決
			char := gg.mangaGenerator.Characters.GetCharacterWithDefault(panel.SpeakerID)
			if char == nil {
				// SpeakerIDが見つからず、デフォルトキャラも存在しない場合はエラーとする
				return fmt.Errorf("character not found for speaker ID '%s' and no default character is available", panel.SpeakerID)
			}

			// プロンプト構築
			userPrompt, systemPrompt, finalSeed := pb.BuildPanelPrompt(panel, char.ID)

			// 構造化ロギングの適用
			logger := slog.With(
				"panel_index", i+1,
				"character_id", char.ID,
				"character_name", char.Name,
				"seed", finalSeed,
			)
			logger.Info("Starting panel generation")

			// 3. アダプター呼び出し
			startTime := time.Now()
			resp, err := gg.mangaGenerator.ImgGen.GenerateMangaPanel(egCtx, imagedom.ImageGenerationRequest{
				Prompt:         userPrompt,
				NegativePrompt: prompts.NegativePanelPrompt,
				SystemPrompt:   systemPrompt,
				Seed:           &finalSeed,
				ReferenceURL:   char.ReferenceURL,
				AspectRatio:    PanelAspectRatio,
			})
			if err != nil {
				return fmt.Errorf("パネル %d (キャラID: %s) の生成に失敗しました: %w", i+1, char.ID, err)
			}

			logger.Info("Panel generation completed",
				"duration", time.Since(startTime).Round(time.Millisecond),
			)

			images[i] = resp
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return images, nil
}
