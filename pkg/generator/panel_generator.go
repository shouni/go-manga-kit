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
)

// PanelGenerator は、キャラクターの一貫性を保ちながら並列で複数パネルを生成します。
type PanelGenerator struct {
	composer *MangaComposer
	pb       prompts.ImagePrompt
}

// NewPanelGenerator は PanelGenerator の新しいインスタンスを初期化します。
func NewPanelGenerator(composer *MangaComposer, pb prompts.ImagePrompt) *PanelGenerator {
	return &PanelGenerator{
		composer: composer,
		pb:       pb,
	}
}

// Execute は、並列処理を用いてパネル群を生成します。
func (pg *PanelGenerator) Execute(ctx context.Context, panels []domain.Panel) ([]*imagedom.ImageResponse, error) {
	if err := pg.composer.PrepareCharacterResources(ctx, panels); err != nil {
		return nil, err
	}

	images := make([]*imagedom.ImageResponse, len(panels))
	eg, egCtx := errgroup.WithContext(ctx)

	cm := pg.composer.CharactersMap

	for i, panel := range panels {
		i, panel := i, panel
		eg.Go(func() error {
			if err := pg.composer.RateLimiter.Wait(egCtx); err != nil {
				return err
			}

			// キャラクター情報を取得
			char := cm.GetCharacterWithDefault(panel.SpeakerID)
			if char == nil {
				return fmt.Errorf("character not found for speaker ID '%s' and no default character is available", panel.SpeakerID)
			}
			finalSeed := char.Seed
			userPrompt, systemPrompt := pg.pb.BuildPanel(panel, char)

			pg.composer.mu.RLock()
			fileURI := pg.composer.CharacterResourceMap[char.ID]
			pg.composer.mu.RUnlock()

			logger := slog.With(
				"panel_index", i+1,
				"character_id", char.ID,
				"character_name", char.Name,
				"seed", finalSeed,
				"use_file_api", fileURI != "",
			)
			logger.Info("Starting panel generation")

			startTime := time.Now()
			resp, err := pg.composer.ImageGenerator.GenerateMangaPanel(egCtx, imagedom.ImageGenerationRequest{
				Prompt:         userPrompt,
				SystemPrompt:   systemPrompt,
				NegativePrompt: prompts.NegativePanelPrompt,
				AspectRatio:    PanelAspectRatio,
				ImageSize:      ImageSize1K,
				Image: imagedom.ImageURI{
					FileAPIURI:   fileURI,
					ReferenceURL: char.ReferenceURL,
				},
				Seed: &finalSeed,
			})
			if err != nil {
				return fmt.Errorf("panel %d (character_id: %s) generation failed: %w", i+1, char.ID, err)
			}

			logger.Info("Panel generation completed", "duration", time.Since(startTime).Round(time.Millisecond))
			images[i] = resp
			return nil
		})
	}

	return images, eg.Wait()
}
