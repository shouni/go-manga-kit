package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/prompt"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// GroupPipeline は、キャラクターの一貫性を保ちながら並列で複数パネルを生成する。
type GroupPipeline struct {
	mangaPipeline Pipeline
	styleSuffix   string
	interval      time.Duration
}

func NewGroupPipeline(mangaPipeline Pipeline, styleSuffix string, interval time.Duration) *GroupPipeline {
	return &GroupPipeline{
		mangaPipeline: mangaPipeline,
		styleSuffix:   styleSuffix,
		interval:      interval,
	}
}

// ExecutePanelGroup は、並列処理を用いてパネル群を生成する。
// ログ出力や進捗管理はここでは行わず、純粋に生成結果を返すことに専念するのだ！
func (gp *GroupPipeline) ExecutePanelGroup(ctx context.Context, pages []domain.MangaPage) ([]*imagedom.ImageResponse, error) {

	pb := prompt.NewPromptBuilder(gp.mangaPipeline.Characters, gp.styleSuffix)

	images := make([]*imagedom.ImageResponse, len(pages))
	eg, egCtx := errgroup.WithContext(ctx)

	// レートリミットの設定。intervalが0なら制限なしとして動くのだ。
	var limiter *rate.Limiter
	if gp.interval > 0 {
		limiter = rate.NewLimiter(rate.Every(gp.interval), 2)
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
			char := gp.resolveAndGetCharacter(page, gp.mangaPipeline.Characters)

			// 2. プロンプト構築
			pmp, negPrompt, finalSeed := pb.BuildUnifiedPrompt(page, page.SpeakerID)

			// 3. シード値の処理
			// char.Seed 自体が設定されているか、あるいは決定論的に生成された finalSeed を使うのだ
			var seedPtr *int64
			seedPtr = &finalSeed

			// 4. アダプター呼び出し
			resp, err := gp.mangaPipeline.ImgGen.GenerateMangaPanel(egCtx, imagedom.ImageGenerationRequest{
				Prompt:         pmp,
				NegativePrompt: negPrompt,
				Seed:           seedPtr,
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

// resolveAndGetCharacter determines and retrieves the appropriate character for a given manga page based on speaker ID or visual cues.
func (gp *GroupPipeline) resolveAndGetCharacter(page domain.MangaPage, characters map[string]domain.Character) domain.Character {
	// 1. IDの正規化
	id := strings.ToLower(strings.TrimSpace(page.SpeakerID))

	// 2. SpeakerIDが空なら、VisualAnchor（描写指示）からキャラ名を推測する
	if id == "" {
		anchor := strings.ToLower(page.VisualAnchor)
		// 優先順位：2人組 -> 特定キャラ の順でチェック
		if strings.Contains(anchor, "zundamon") && strings.Contains(anchor, "metan") {
			id = "zundamon_metan"
		} else if strings.Contains(anchor, "metan") {
			id = "metan"
		} else if strings.Contains(anchor, "zundamon") {
			id = "zundamon"
		}
	}

	// 3. 確定したIDで検索
	if c, ok := characters[id]; ok {
		return c
	}

	// 4. 見つからない場合のフォールバック（ここは慎重に！）
	// IDが指定されていたのに見つからない場合、勝手に「ずんだもん」にするよりは、
	// 名前だけ入れた空のCharacterを返して BuildUnifiedPrompt 側の GetSeedFromName に任せるのが安全なのだ。
	if id != "" {
		return domain.Character{ID: id, Name: id}
	}

	// 5. 本当に何も情報がない場合のみ、デフォルトキャラを返す
	if zunda, ok := characters["zundamon"]; ok {
		return zunda
	}

	// 最終手段
	slog.Warn("Could not resolve character from page info, falling back to 'Unknown'", "speakerID", page.SpeakerID, "visualAnchor", page.VisualAnchor)
	return domain.Character{Name: "Unknown"}
}
