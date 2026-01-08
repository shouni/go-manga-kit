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
}

// NewGroupGenerator は GroupGenerator の新しいインスタンスを初期化します。
func NewGroupGenerator(mangaGenerator MangaGenerator, styleSuffix string, interval time.Duration) *GroupGenerator {
	return &GroupGenerator{
		mangaGenerator: mangaGenerator,
		styleSuffix:    styleSuffix,
		interval:       interval,
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

			// 2. プロンプト構築（決定論的な Seed 値生成を含む）
			pmp, negPrompt, finalSeed := pb.BuildUnifiedPrompt(page, page.SpeakerID)

			// 3. アダプター呼び出し
			// シード値は BuildUnifiedPrompt で計算されたアドレスを直接渡すのだ
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

// resolveAndGetCharacter は、与えられたページ情報とキャラクターリストから、最も適切と思われるキャラクターを解決して返します。
// 解決ロジックは以下の優先順位で実行されます:
// 1. ページに指定された SpeakerID に完全一致するキャラクター。
// 2. IsPrimary フラグが true に設定されているキャラクター（複数ある場合は ID 順で最初）。
// 3. 上記で見つからない場合、キャラクターリストを ID でソートした際の最初のキャラクター。
// 4. キャラクターリストが空の場合、"Unknown" という名前のデフォルトキャラクター。
func (gp *GroupGenerator) resolveAndGetCharacter(page domain.MangaPage, characters map[string]domain.Character) domain.Character {
	// 1. IDでの直接検索
	id := strings.ToLower(strings.TrimSpace(page.SpeakerID))
	if c, ok := characters[id]; ok {
		return c
	}

	// 2. マップの非決定性を排除するためのソート済みスライスの作成
	keys := make([]string, 0, len(characters))
	for k := range characters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 3. ソートされた順序で Primary キャラクターを探索
	for _, k := range keys {
		if char := characters[k]; char.IsPrimary {
			return char // 最初に見つかった（＝ID順で最小の）Primaryを返す
		}
	}

	// 4. Primary が見つからなかった場合、ソート順の最初のキャラをフォールバックとして返す
	if len(keys) > 0 {
		fallbackID := keys[0]
		slog.Debug("Primary character not found, falling back to deterministic first character",
			"originalID", page.SpeakerID, "selectedID", fallbackID)
		return characters[fallbackID]
	}

	// 5. 最終手段 (キャラクターマップが空の場合)
	slog.Warn("No characters available in the map", "speakerID", page.SpeakerID)
	return domain.Character{Name: "Unknown"}
}
