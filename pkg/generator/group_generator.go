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
	styleSuffix    string
	interval       time.Duration
	sortedCharKeys []string          // 決定論的な解決のために事前にソートされたIDリスト
	primaryChar    *domain.Character // 優先的にフォールバック先となる Primary キャラクター
}

// NewGroupGenerator は GroupGenerator の新しいインスタンスを初期化します。
// キャラクターマップの解析（ソート・Primary特定の事前計算）を行い、生成時のコストを最適化します。
func NewGroupGenerator(mangaGenerator MangaGenerator, styleSuffix string, interval time.Duration) *GroupGenerator {
	keys := make([]string, 0, len(mangaGenerator.Characters))
	for k := range mangaGenerator.Characters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 最も優先度の高い（ID順で最小の）Primaryキャラクターを事前に特定しておく
	var primary *domain.Character
	for _, k := range keys {
		if char := mangaGenerator.Characters[k]; char.IsPrimary {
			primary = &char // このスコープではcharが再代入される前にbreakするため安全です
			break
		}
	}

	return &GroupGenerator{
		mangaGenerator: mangaGenerator,
		styleSuffix:    styleSuffix,
		interval:       interval,
		sortedCharKeys: keys,
		primaryChar:    primary,
	}
}

// ExecutePanelGroup は、並列処理を用いてパネル群を生成します。
func (gg *GroupGenerator) ExecutePanelGroup(ctx context.Context, pages []domain.MangaPage) ([]*imagedom.ImageResponse, error) {
	pb := prompts.NewImagePromptBuilder(gg.mangaGenerator.Characters, gg.styleSuffix)
	images := make([]*imagedom.ImageResponse, len(pages))
	eg, egCtx := errgroup.WithContext(ctx)

	var limiter *rate.Limiter
	if gg.interval > 0 {
		// APIのバースト制限を考慮し、同時に2リクエストまでを許容するレートリミッターを設定します。
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

			// 1. キャラクター解決（不正な SpeakerID や空の場合に備える）
			char := gg.resolveAndGetCharacter(page, gg.mangaGenerator.Characters)

			// 2. プロンプト構築
			pmp, negPrompt, finalSeed := pb.BuildUnifiedPrompt(page, char.ID)

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

// resolveAndGetCharacter は、与えられたページ情報から最適なキャラクターを決定します。
// 解決ロジックは以下の優先順位で実行されます:
// 1. ページに指定された SpeakerID に完全一致するキャラクター。
// 2. IsPrimary フラグが true に設定されているキャラクター（NewGroupGenerator で事前特定済み）。
// 3. 上記で見つからない場合、事前にソートされた ID リストの最初のキャラクター。
// 4. キャラクターリストが空の場合、"Unknown" という名前のデフォルトキャラクター。
func (gp *GroupGenerator) resolveAndGetCharacter(page domain.MangaPage, characters map[string]domain.Character) domain.Character {
	// 1. IDでの直接検索
	id := strings.ToLower(strings.TrimSpace(page.SpeakerID))
	if c, ok := characters[id]; ok {
		return c
	}

	if id != "" {
		slog.Debug("SpeakerID not found in character map, attempting to resolve fallback", "speakerID", id)
	}

	// 2. 事前に特定した Primary キャラクターを優先フォールバック
	if gp.primaryChar != nil {
		return *gp.primaryChar
	}

	// 3. Primary がいない場合、ソート順の最初のキャラをフォールバック
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
