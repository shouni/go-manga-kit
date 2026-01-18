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
	return &PanelGenerator{
		composer: composer,
	}
}

// Execute は、並列処理を用いてパネル群を生成します。
func (pg *PanelGenerator) Execute(ctx context.Context, panels []domain.Panel) ([]*imagedom.ImageResponse, error) {

	// 1. リソースの事前準備（別の関数に切り出し）
	if err := pg.prepareCharacterResources(ctx, panels); err != nil {
		return nil, err
	}

	images := make([]*imagedom.ImageResponse, len(panels))
	eg, egCtx := errgroup.WithContext(ctx)

	pb := pg.composer.PromptBuilder
	cm := pg.composer.CharactersMap

	// --- 2. パネル生成フェーズ (並列実行) ---
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

			// 事前準備した File API URI を取得（存在しない場合は空文字となりフォールバックされる）
			fileURI := pg.composer.characterResourceMap[char.ID]

			logger := slog.With(
				"panel_index", i+1,
				"speaker_id", char.ID,
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
				return fmt.Errorf("panel %d (speaker: %s) generation failed: %w", i+1, char.ID, err)
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

// prepareCharacterResources はパネルに使用される全キャラクターの画像を File API に事前アップロードします。
func (pg *PanelGenerator) prepareCharacterResources(ctx context.Context, panels []domain.Panel) error {
	if pg.composer.characterResourceMap == nil {
		pg.composer.characterResourceMap = make(map[string]string)
	}

	uniqueSpeakerIDs := ExtractUniqueSpeakerIDs(panels)
	cm := pg.composer.CharactersMap

	for _, speakerID := range uniqueSpeakerIDs {
		// すでに URI 保持済みならスキップ
		if _, ok := pg.composer.characterResourceMap[speakerID]; ok {
			continue
		}

		char := cm.GetCharacterWithDefault(speakerID)
		if char == nil || char.ReferenceURL == "" {
			continue
		}

		slog.Info("Uploading character asset to Gemini File API",
			"speaker_id", speakerID,
			"url", char.ReferenceURL,
		)

		uri, err := pg.composer.AssetManager.UploadFile(ctx, char.ReferenceURL)
		if err != nil {
			return fmt.Errorf("failed to upload asset for speaker %s: %w", speakerID, err)
		}

		pg.composer.characterResourceMap[speakerID] = uri
	}
	return nil
}

// ExtractUniqueSpeakerIDs はパネルのスライスから重複しない SpeakerID を抽出します。
func ExtractUniqueSpeakerIDs(panels []domain.Panel) []string {

	set := make(map[string]struct{})

	for _, panel := range panels {
		// 空文字でない場合のみ追加
		if panel.SpeakerID != "" {
			set[panel.SpeakerID] = struct{}{}
		}
	}

	// 抽出されたIDをスライスに変換
	uniqueIDs := make([]string, 0, len(set))
	for id := range set {
		uniqueIDs = append(uniqueIDs, id)
	}

	return uniqueIDs
}
