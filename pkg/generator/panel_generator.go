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

			// キャラクター解決（デフォルトキャラへのフォールバック含む）
			char := cm.GetCharacterWithDefault(panel.SpeakerID)
			if char == nil {
				return fmt.Errorf("character not found for speaker ID '%s' and no default character is available", panel.SpeakerID)
			}

			userPrompt, systemPrompt, finalSeed := pb.BuildPanelPrompt(panel, char.ID)

			// 解決後の char.ID をキーに URI を取得
			pg.composer.mu.RLock()
			fileURI := pg.composer.characterResourceMap[char.ID]
			pg.composer.mu.RUnlock()

			// ログフィールド名を character_id に修正し、実体と一致させる
			logger := slog.With(
				"panel_index", i+1,
				"character_id", char.ID,
				"seed", finalSeed,
				"use_file_api", fileURI != "",
			)

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
				return fmt.Errorf("panel %d (character_id: %s) generation failed: %w", i+1, char.ID, err)
			}

			logger.Info("Panel generation completed", "duration", time.Since(startTime).Round(time.Millisecond))
			images[i] = resp
			return nil
		})
	}

	return images, eg.Wait()
}

// prepareCharacterResources はパネルに使用される全キャラクターの画像を File API に事前アップロードします。
// singleflight を利用して、同一キャラクターの重複アップロード（APIコール）を完全に阻止します。
func (pg *PanelGenerator) prepareCharacterResources(ctx context.Context, panels []domain.Panel) error {
	pg.composer.mu.Lock()
	if pg.composer.characterResourceMap == nil {
		pg.composer.characterResourceMap = make(map[string]string)
	}
	pg.composer.mu.Unlock()

	uniqueSpeakerIDs := domain.Panels(panels).UniqueSpeakerIDs()
	cm := pg.composer.CharactersMap
	eg, egCtx := errgroup.WithContext(ctx)

	for _, id := range uniqueSpeakerIDs {
		speakerID := id

		eg.Go(func() error {
			// キャラクター解決
			char := cm.GetCharacterWithDefault(speakerID)
			if char == nil || char.ReferenceURL == "" {
				return nil
			}
			resolvedCharID := char.ID

			// singleflight.Do により、同じ resolvedCharID に対する処理が複数走っている場合、
			// 最初の1つだけが実行され、他はその完了（結果）を待ちます。
			_, err, _ := pg.composer.uploadGroup.Do(resolvedCharID, func() (interface{}, error) {
				// 1. 既に他の singleflight 呼び出しによって完了しているかチェック
				pg.composer.mu.RLock()
				uri, ok := pg.composer.characterResourceMap[resolvedCharID]
				pg.composer.mu.RUnlock()
				if ok {
					return uri, nil
				}

				// 2. アップロード実行（重い I/O 処理）
				// ロックの外で実行されるため、異なるキャラクターのアップロードは並列に進みます
				uploadedURI, uploadErr := pg.composer.AssetManager.UploadFile(egCtx, char.ReferenceURL)
				if uploadErr != nil {
					return nil, uploadErr
				}

				// 3. マップに書き込み
				pg.composer.mu.Lock()
				pg.composer.characterResourceMap[resolvedCharID] = uploadedURI
				pg.composer.mu.Unlock()

				return uploadedURI, nil
			})

			if err != nil {
				return fmt.Errorf("failed to prepare asset for character %s (resolved from speaker %s): %w", resolvedCharID, speakerID, err)
			}

			return nil
		})
	}

	return eg.Wait()
}
