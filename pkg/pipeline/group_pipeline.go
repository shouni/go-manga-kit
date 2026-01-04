package pipeline

import (
	"context"
	"fmt"
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
			pmp, negPrompt, seed := pb.BuildUnifiedPrompt(page, page.SpeakerID)
			//			prompt, negPrompt := gp.buildPrompt(page.VisualAnchor, char.VisualCues)

			// 3. シード値の処理 (int64 -> *int64)
			var seedPtr *int64
			if char.Seed > 0 {
				s := seed
				seedPtr = &s
			}

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

// 以下、内部ロジック（Runnerから移管したもの）

func (gp *GroupPipeline) resolveAndGetCharacter(page domain.MangaPage, characters map[string]domain.Character) domain.Character {
	id := strings.ToLower(page.SpeakerID)
	if id == "" {
		anchor := strings.ToLower(page.VisualAnchor)
		if strings.Contains(anchor, "metan") {
			id = "metan"
		}
		if strings.Contains(anchor, "zundamon") {
			id = "zundamon"
		}
	}

	if c, ok := characters[id]; ok {
		return c
	}
	if zunda, ok := characters["zundamon"]; ok {
		return zunda
	}
	for _, v := range characters {
		return v
	}
	return domain.Character{Name: "Unknown"}
}

//// TODO::あとで消す
//func (gp *GroupPipeline) buildPrompt(anchor string, cues []string) (string, string) {
//	positive := fmt.Sprintf("%s, %s, %s, cinematic composition, high resolution, no speech bubbles",
//		gp.basePrompt, strings.Join(cues, ", "), anchor)
//	negative := "speech bubble, dialogue balloon, text, alphabet, letters, words, signatures, watermark, username, low quality, distorted, bad anatomy"
//	return positive, negative
//}
