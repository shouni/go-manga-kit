package generator

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
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
	interval       time.Duration
	sortedCharKeys []string          // 決定論的な解決のために事前にソートされたIDリスト
	primaryChar    *domain.Character // 優先的にフォールバック先となる Primary キャラクター
}

// NewGroupGenerator は GroupGenerator の新しいインスタンスを初期化します。
// キャラクターマップの解析（ソート・Primary特定の事前計算）を行い、生成時のコストを最適化します。
func NewGroupGenerator(mangaGenerator MangaGenerator, interval time.Duration) *GroupGenerator {
	keys := make([]string, 0, len(mangaGenerator.Characters))
	for k := range mangaGenerator.Characters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var primary *domain.Character
	for _, k := range keys {
		if char := mangaGenerator.Characters[k]; char.IsPrimary {
			c := char // ループ変数のアドレス回避
			primary = &c
			break
		}
	}

	return &GroupGenerator{
		mangaGenerator: mangaGenerator,
		interval:       interval,
		sortedCharKeys: keys,
		primaryChar:    primary,
	}
}

// ExecutePanelGroup は、並列処理を用いてパネル群を生成します。
func (gg *GroupGenerator) ExecutePanelGroup(ctx context.Context, pages []domain.MangaPage) ([]*imagedom.ImageResponse, error) {
	// プロンプトビルダーの初期化
	pb := gg.mangaGenerator.PromptBuilder
	images := make([]*imagedom.ImageResponse, len(pages))
	eg, egCtx := errgroup.WithContext(ctx)

	var limiter *rate.Limiter
	if gg.interval > 0 {
		// APIのバースト制限を考慮し、レートリミッターを設定します。
		limiter = rate.NewLimiter(rate.Every(gg.interval), 2)
	}

	for i, page := range pages {
		i, page := i, page
		eg.Go(func() error {
			if limiter != nil {
				if err := limiter.Wait(egCtx); err != nil {
					return err
				}
			}

			// 1. キャラクター解決
			char := gg.resolveAndGetCharacter(page, gg.mangaGenerator.Characters)

			// 2. プロンプト構築
			userPrompt, systemPrompt, finalSeed := pb.BuildPanelPrompt(page, char.ID)

			slog.Info("パネル生成開始",
				"panel_index", i+1,
				"character_id", char.ID,
				"seed", finalSeed)

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
				return fmt.Errorf("パネル %d (キャラID: %s, 名前: %s) の生成に失敗しました: %w", i+1, char.ID, char.Name, err)
			}

			slog.Info("パネル生成完了",
				"panel_index", i+1,
				"character", char.Name,
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

// resolveAndGetCharacter は、与えられたページ情報から最適なキャラクターを決定します。
func (gg *GroupGenerator) resolveAndGetCharacter(page domain.MangaPage, characters map[string]domain.Character) domain.Character {
	// 1. IDでの直接検索
	id := strings.ToLower(strings.TrimSpace(page.SpeakerID))
	if c, ok := characters[id]; ok {
		return c
	}

	if id != "" {
		slog.Debug("SpeakerID がマップに見つかりません。フォールバックを試みます", "speakerID", id)
	}

	// 2. 事前に特定した Primary キャラクターを優先フォールバック
	if gg.primaryChar != nil {
		return *gg.primaryChar
	}

	// 3. Primary がいない場合、ソート順の最初のキャラをフォールバック
	if len(gg.sortedCharKeys) > 0 {
		fallbackID := gg.sortedCharKeys[0]
		slog.Debug("Primary キャラクター不在のため、決定論的な最初のキャラを採用します",
			"originalID", page.SpeakerID, "selectedID", fallbackID)
		return characters[fallbackID]
	}

	// 4. 最終手段
	slog.Warn("利用可能なキャラクターが定義されていません", "speakerID", page.SpeakerID)
	return domain.Character{Name: "Unknown"}
}
