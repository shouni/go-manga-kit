package generator

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
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
	primaryChar    *domain.Character // 優先的にフォールバック先となる Primary キャラクター
	sortedCharKeys []string          // 決定論的な解決のために事前にソートされたIDリスト
}

// NewGroupGenerator は GroupGenerator の新しいインスタンスを初期化します。
func NewGroupGenerator(mangaGenerator MangaGenerator, interval time.Duration) *GroupGenerator {
	keys := make([]string, 0, len(mangaGenerator.Characters))
	for k := range mangaGenerator.Characters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return &GroupGenerator{
		mangaGenerator: mangaGenerator,
		limiter:        rate.NewLimiter(rate.Every(interval), 2),
		primaryChar:    mangaGenerator.Characters.GetPrimary(),
		sortedCharKeys: keys,
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

			// 1. キャラクター解決
			char := gg.resolveAndGetCharacter(panel)

			// 2. プロンプト構築 (最新の BuildPanelPrompt 仕様に合わせる)
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
				NegativePrompt: prompts.DefaultNegativePanelPrompt,
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

// resolveAndGetCharacter は、与えられたページ情報から最適なキャラクターを決定します。
func (gg *GroupGenerator) resolveAndGetCharacter(panel domain.Panel) domain.Character {
	charMap := gg.mangaGenerator.Characters
	// 1. IDでの直接検索
	char := charMap.FindCharacter(panel.SpeakerID)
	if char != nil {
		return *char
	}

	slog.Debug("SpeakerID not found in map, attempting fallback", "speaker_id", panel.SpeakerID)

	// 2. 事前に特定した Primary キャラクターを優先フォールバック
	if gg.primaryChar != nil {
		return *gg.primaryChar
	}

	// 3. Primary がいない場合、ソート順の最初のキャラをフォールバック
	if len(gg.sortedCharKeys) > 0 {
		fallbackID := gg.sortedCharKeys[0]
		return charMap[fallbackID]
	}

	// 4. 最終手段
	return domain.Character{ID: "unknown", Name: "Unknown"}
}
