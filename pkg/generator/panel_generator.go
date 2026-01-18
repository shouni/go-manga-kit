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

func NewPanelGenerator(composer *MangaComposer) *PanelGenerator {
	return &PanelGenerator{composer: composer}
}

func (pg *PanelGenerator) Execute(ctx context.Context, panels []domain.Panel) ([]*imagedom.ImageResponse, error) {
	// 1. リソースの事前準備（スレッドセーフ化済み）
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
				return fmt.Errorf("character not found for speaker ID '%s'", panel.SpeakerID)
			}

			userPrompt, systemPrompt, finalSeed := pb.BuildPanelPrompt(panel, char.ID)

			// 読み取りもロックをかけて安全に取得
			pg.composer.mu.Lock()
			fileURI := pg.composer.characterResourceMap[char.ID]
			pg.composer.mu.Unlock()

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

// prepareCharacterResources パネル生成中にキャラクターの一貫性を保つために必要なリソースを準備します。
// キャラクターリソースマップのスレッドセーフな初期化を保証し、必要に応じてアセットをアップロードします。
// アセットのアップロードに失敗した場合はエラーを返します。
func (pg *PanelGenerator) prepareCharacterResources(ctx context.Context, panels []domain.Panel) error {
	// マップ初期化時の保護
	pg.composer.mu.Lock()
	if pg.composer.characterResourceMap == nil {
		pg.composer.characterResourceMap = make(map[string]string)
	}
	pg.composer.mu.Unlock()

	// domain.Panels 型へのキャストによるメソッド呼び出し
	uniqueSpeakerIDs := domain.Panels(panels).UniqueSpeakerIDs()
	cm := pg.composer.CharactersMap

	for _, speakerID := range uniqueSpeakerIDs {
		// 存在確認のロック
		pg.composer.mu.Lock()
		_, ok := pg.composer.characterResourceMap[speakerID]
		pg.composer.mu.Unlock()
		if ok {
			continue
		}

		char := cm.GetCharacterWithDefault(speakerID)
		if char == nil || char.ReferenceURL == "" {
			continue
		}

		// API 呼び出し（時間のかかる処理）はロックの外で行う
		uri, err := pg.composer.AssetManager.UploadFile(ctx, char.ReferenceURL)
		if err != nil {
			return fmt.Errorf("failed to upload asset for speaker %s: %w", speakerID, err)
		}

		// 書き込みのロック
		pg.composer.mu.Lock()
		pg.composer.characterResourceMap[speakerID] = uri
		pg.composer.mu.Unlock()
	}
	return nil
}
