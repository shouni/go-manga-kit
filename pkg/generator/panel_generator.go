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
	// リソースの事前準備（並列アップロードを実行）
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
				return fmt.Errorf("character not found for speaker ID '%s' and no default character is available", panel.SpeakerID)
			}

			userPrompt, systemPrompt, finalSeed := pb.BuildPanelPrompt(panel, char.ID)

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
				// エラーメッセージに speaker_id を含めてデバッグ効率を向上
				return fmt.Errorf("panel %d (speaker_id: %s) generation failed: %w", i+1, char.ID, err)
			}

			logger.Info("Panel generation completed", "duration", time.Since(startTime).Round(time.Millisecond))
			images[i] = resp
			return nil
		})
	}

	return images, eg.Wait()
}

// prepareCharacterResources はパネルに使用される全キャラクターの画像を File API に事前アップロードします。
// 各キャラクターのアップロード処理を errgroup により並列で実行し、ロックの保持時間を最小化します。
func (pg *PanelGenerator) prepareCharacterResources(ctx context.Context, panels []domain.Panel) error {
	// マップの遅延初期化
	pg.composer.mu.Lock()
	if pg.composer.characterResourceMap == nil {
		pg.composer.characterResourceMap = make(map[string]string)
	}
	pg.composer.mu.Unlock()

	uniqueSpeakerIDs := domain.Panels(panels).UniqueSpeakerIDs()
	cm := pg.composer.CharactersMap
	eg, egCtx := errgroup.WithContext(ctx)

	for _, id := range uniqueSpeakerIDs {
		speakerID := id // ループ変数のキャプチャ

		eg.Go(func() error {
			// まず Read ロックで存在確認
			pg.composer.mu.RLock()
			_, ok := pg.composer.characterResourceMap[speakerID]
			pg.composer.mu.RUnlock()
			if ok {
				return nil
			}

			char := cm.GetCharacterWithDefault(speakerID)
			if char == nil || char.ReferenceURL == "" {
				return nil
			}

			// 時間のかかる API 呼び出しはロックの外で実行（並列 I/O を許容）
			uri, err := pg.composer.AssetManager.UploadFile(egCtx, char.ReferenceURL)
			if err != nil {
				return fmt.Errorf("failed to upload asset for speaker %s: %w", speakerID, err)
			}

			// 書き込み時に Lock を取得し、再度存在確認（ダブルチェック）
			pg.composer.mu.Lock()
			defer pg.composer.mu.Unlock()
			if _, ok := pg.composer.characterResourceMap[speakerID]; !ok {
				pg.composer.characterResourceMap[speakerID] = uri
			}
			return nil
		})
	}

	return eg.Wait()
}
