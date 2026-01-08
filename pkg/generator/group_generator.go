package generator

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/prompt"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// GroupGenerator は、キャラクターの一貫性を保ちながら並列で複数パネルを生成します。
type GroupGenerator struct {
	mangaGenerator MangaGenerator
	styleSuffix    string
	interval       time.Duration
	sortedCharKeys []string
}

// NewGroupGenerator は GroupGenerator の新しいインスタンスを初期化します。
// キャラクターマップのキーを事前にソートして保持することで、生成時の計算コストを削減します。
func NewGroupGenerator(mangaGenerator MangaGenerator, styleSuffix string, interval time.Duration) *GroupGenerator {
	keys := make([]string, 0, len(mangaGenerator.Characters))
	for k := range mangaGenerator.Characters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return &GroupGenerator{
		mangaGenerator: mangaGenerator,
		styleSuffix:    styleSuffix,
		interval:       interval,
		sortedCharKeys: keys,
	}
}

// ExecutePanelGroup は、並列処理を用いてパネル群を生成します。
func (gg *GroupGenerator) ExecutePanelGroup(ctx context.Context, pages []domain.MangaPage) ([]*imagedom.ImageResponse, error) {
	pb := prompt.NewPromptBuilder(gg.mangaGenerator.Characters, gg.styleSuffix)
	images := make([]*imagedom.ImageResponse, len(pages))
	eg, egCtx := errgroup.WithContext(ctx)

	var limiter *rate.Limiter
	if gg.interval > 0 {
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
			pmp, negPrompt, finalSeed := pb.BuildUnifiedPrompt(page, page.SpeakerID)

			// 3. アダプター呼び出し
			resp, err := gg.mangaGenerator.ImgGen.GenerateMangaPanel(egCtx, imagedom.ImageGenerationRequest{
				Prompt:         pmp,
				NegativePrompt: negPrompt,
				Seed:           &finalSeed,
				ReferenceURL:   char.ReferenceURL,
				AspectRatio:    "16:9",
			})
			if err != nil {
				return fmt.Errorf("page %d (char: %s) generation failed: %w", i+1, char.Name, err)
			}

			images[i] = resp
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return images, nil
}

// resolveAndGetCharacter は、与えられたページ情報とキャラクターリストから最も適切なキャラクターを解決します。
// 解決ロジックは以下の優先順位で実行されます:
// 1. SpeakerID に完全一致する ID を持つキャラクター。
// 2. IsPrimary フラグが true に設定されているキャラクター（複数ある場合は ID 順で最初）。
// 3. 上記で見つからない場合、キャッシュされた ID リストの最初のキャラクター。
// 4. キャラクターリストが空の場合、"Unknown" という名前のデフォルトキャラクター。
func (gp *GroupGenerator) resolveAndGetCharacter(page domain.MangaPage, characters map[string]domain.Character) domain.Character {
	// 1. IDでの直接検索
	id := strings.ToLower(strings.TrimSpace(page.SpeakerID))
	if c, ok := characters[id]; ok {
		return c
	}

	// 指定されたIDが見つからない場合のログ記録
	if id != "" {
		slog.Debug("SpeakerID not found in character map, attempting to resolve fallback", "speakerID", id)
	}

	// 2. ソート済みキャッシュを用いて Primary キャラクターを探索
	for _, k := range gp.sortedCharKeys {
		if char := characters[k]; char.IsPrimary {
			return char // ID順で最小の Primary を返す
		}
	}

	// 3. Primary が見つからなかった場合、ソート順の最初のキャラをフォールバックとして返す
	if len(gp.sortedCharKeys) > 0 {
		fallbackID := gp.sortedCharKeys[0]
		slog.Debug("Primary character not found, falling back to deterministic first character",
			"originalID", page.SpeakerID, "selectedID", fallbackID)
		return characters[fallbackID]
	}

	// 4. 最終手段
	slog.Warn("No characters available in the map", "speakerID", page.SpeakerID)
	return domain.Character{Name: "Unknown"}
}
