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
}

// NewPanelGenerator は PanelGenerator の新しいインスタンスを初期化します。
func NewPanelGenerator(composer *MangaComposer) *PanelGenerator {
	return &PanelGenerator{composer: composer}
}

// Execute は、並列処理を用いてパネル群を生成します。
// 事前にキャラクターリソースを準備し、各パネルの画像生成を並行して実行します。
func (pg *PanelGenerator) Execute(ctx context.Context, panels []domain.Panel) ([]*imagedom.ImageResponse, error) {
	// 1. リソースの事前準備
	if err := pg.prepareCharacterResources(ctx, panels); err != nil {
		return nil, err
	}

	images := make([]*imagedom.ImageResponse, len(panels))
	eg, egCtx := errgroup.WithContext(ctx)

	pb := pg.composer.PromptBuilder
	cm := pg.composer.CharactersMap

	for i, panel := range panels {
		i, panel := i, panel
		eg.Go(func() error {
			if err := pg.composer.RateLimiter.Wait(egCtx); err != nil {
				return err
			}

			char := cm.GetCharacterWithDefault(panel.SpeakerID)
			if char == nil {
				// エラーメッセージを詳細化：デフォルトキャラも不在であることを明記
				return fmt.Errorf("character not found for speaker ID '%s' and no default character is available", panel.SpeakerID)
			}

			userPrompt, systemPrompt, finalSeed := pb.BuildPanelPrompt(panel, char.ID)

			// 読み取りは RLock (Read Lock) で並列実行を許可
			pg.composer.mu.RLock()
			fileURI := pg.composer.characterResourceMap[char.ID]
			pg.composer.mu.RUnlock()

			logger := slog.With("panel_index", i+1, "speaker_id", char.ID, "use_file_api", fileURI != "")
			logger.Info("Starting panel generation")

			startTime := time.Now()
			resp, err := pg.composer.ImageGenerator.GenerateMangaPanel(egCtx, imagedom.ImageGenerationRequest{
				Prompt:         userPrompt,
				NegativePrompt: prompts.NegativePanelPrompt,
				SystemPrompt:   systemPrompt,
				Seed:           &finalSeed,
				FileAPIURI:     fileURI,
				ReferenceURL:   char.ReferenceURL,
				AspectRatio:    PanelAspectRatio,
			})
			if err != nil {
				return fmt.Errorf("panel %d generation failed: %w", i+1, err)
			}

			logger.Info("Panel generation completed", "duration", time.Since(startTime).Round(time.Millisecond))
			images[i] = resp
			return nil
		})
	}

	return images, eg.Wait()
}

// prepareCharacterResources はパネルに使用される全キャラクターの画像を File API に事前アップロードします。
// 二重アップロードを防ぐため、チェックと書き込みをアトミックに実行します。
func (pg *PanelGenerator) prepareCharacterResources(ctx context.Context, panels []domain.Panel) error {
	pg.composer.mu.Lock()
	if pg.composer.characterResourceMap == nil {
		pg.composer.characterResourceMap = make(map[string]string)
	}
	pg.composer.mu.Unlock()

	uniqueSpeakerIDs := domain.Panels(panels).UniqueSpeakerIDs()
	cm := pg.composer.CharactersMap

	for _, speakerID := range uniqueSpeakerIDs {
		char := cm.GetCharacterWithDefault(speakerID)
		if char == nil || char.ReferenceURL == "" {
			continue
		}

		// 二重アップロード（競合）を防止するため、存在確認から書き込みまでを Lock で保護
		pg.composer.mu.Lock()
		if _, ok := pg.composer.characterResourceMap[speakerID]; ok {
			pg.composer.mu.Unlock()
			continue
		}

		// 時間のかかる API 呼び出し中もロックを保持することで、同一 speakerID の重複リクエストを完全に阻止
		uri, err := pg.composer.AssetManager.UploadFile(ctx, char.ReferenceURL)
		if err != nil {
			pg.composer.mu.Unlock()
			return fmt.Errorf("failed to upload asset for speaker %s: %w", speakerID, err)
		}

		pg.composer.characterResourceMap[speakerID] = uri
		pg.composer.mu.Unlock()
	}
	return nil
}
