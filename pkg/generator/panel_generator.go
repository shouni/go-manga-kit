package generator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	imagePorts "github.com/shouni/gemini-image-kit/pkg/ports"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"golang.org/x/sync/errgroup"
)

// negativePanelPrompt は単体パネルで「文字」や「フキダシ」を徹底排除するための指定です。
const negativePanelPrompt = "speech bubble, dialogue balloon, text, alphabet, letters, words, signatures, watermark, username, low quality, distorted, bad anatomy, monochrome, black and white, greyscale"

// PanelGenerator は、キャラクターの一貫性を保ちながら並列で複数パネルを生成します。
type PanelGenerator struct {
	composer *MangaComposer
	pb       domain.ImagePrompt
}

// NewPanelGenerator は PanelGenerator の新しいインスタンスを初期化します。
func NewPanelGenerator(composer *MangaComposer, pb domain.ImagePrompt) *PanelGenerator {
	return &PanelGenerator{
		composer: composer,
		pb:       pb,
	}
}

// Execute は、errgroupの制限機能を使用して同時実行数を制限しながらパネルを並列生成します。
func (pg *PanelGenerator) Execute(ctx context.Context, panels []domain.Panel) ([]*imagePorts.ImageResponse, error) {
	if len(panels) == 0 {
		return nil, nil
	}

	if err := pg.composer.PrepareCharacterResources(ctx, panels); err != nil {
		return nil, err
	}

	images := make([]*imagePorts.ImageResponse, len(panels))
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(int(pg.composer.MaxConcurrency))

	cm := pg.composer.CharactersMap

	for i, panel := range panels {
		eg.Go(func() error {
			if err := pg.composer.RateLimiter.Wait(egCtx); err != nil {
				return err
			}

			char := cm.GetCharacterWithDefault(panel.SpeakerID)
			if char == nil {
				return fmt.Errorf("character not found for speaker ID '%s'", panel.SpeakerID)
			}
			seed := char.Seed
			userPrompt, systemPrompt := pg.pb.BuildPanel(panel, char)
			fileURI := pg.composer.GetCharacterResourceURI(char.ID)

			logger := slog.With(
				"panel_index", i+1,
				"character_id", char.ID,
				"character_name", char.Name,
				"seed", seed,
				"use_file_api", fileURI != "",
			)
			logger.Info("Starting panel generation")

			startTime := time.Now()
			resp, err := pg.composer.ImageGenerator.GenerateMangaPanel(egCtx, imagePorts.ImagePanelRequest{
				GenerationOptions: imagePorts.GenerationOptions{
					Prompt:         userPrompt,
					SystemPrompt:   systemPrompt,
					NegativePrompt: negativePanelPrompt,
					AspectRatio:    PanelAspectRatio,
					ImageSize:      ImageSize1K,
					Seed:           &seed,
				},
				Image: imagePorts.ImageURI{
					FileAPIURI:   fileURI,
					ReferenceURL: char.ReferenceURL,
				},
			})
			if err != nil {
				return fmt.Errorf("panel %d (character_id: %s) generation failed: %w", i+1, char.ID, err)
			}

			logger.Info("Panel generation completed",
				"duration", time.Since(startTime).Round(time.Second),
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
