package generator

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

// GroupGenerator は、キャラクターの一貫性を保ちながら並列で複数パネルを生成する。
type GroupGenerator struct {
	mangaGenerator MangaGenerator
	styleSuffix    string
	interval       time.Duration
}

func NewGroupGenerator(mangaGenerator MangaGenerator, styleSuffix string, interval time.Duration) *GroupGenerator {
	return &GroupGenerator{
		mangaGenerator: mangaGenerator,
		styleSuffix:    styleSuffix,
		interval:       interval,
	}
}

// ExecutePanelGroup は、並列処理を用いてパネル群を生成する。
// ログ出力や進捗管理はここでは行わず、純粋に生成結果を返すことに専念するのだ！
func (gg *GroupGenerator) ExecutePanelGroup(ctx context.Context, pages []domain.MangaPage) ([]*imagedom.ImageResponse, error) {

	pb := prompt.NewPromptBuilder(gg.mangaGenerator.Characters, gg.styleSuffix)

	images := make([]*imagedom.ImageResponse, len(pages))
	eg, egCtx := errgroup.WithContext(ctx)

	// レートリミットの設定。intervalが0なら制限なしとして動くのだ。
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

			// 3. シード値の処理
			// char.Seed 自体が設定されているか、あるいは決定論的に生成された finalSeed を使うのだ
			var seedPtr *int64
			seedPtr = &finalSeed

			// 4. アダプター呼び出し
			resp, err := gg.mangaGenerator.ImgGen.GenerateMangaPanel(egCtx, imagedom.ImageGenerationRequest{
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

// resolveAndGetCharacter は、ページ情報から最適なキャラクターを決定します。
// IDで特定できない場合は、IsPrimary フラグが立っているキャラクターを優先的に返します。
func (gp *GroupGenerator) resolveAndGetCharacter(page domain.MangaPage, characters map[string]domain.Character) domain.Character {
	// 1. IDでの直接検索（正規化して一致を確認）
	id := strings.ToLower(strings.TrimSpace(page.SpeakerID))
	if c, ok := characters[id]; ok {
		return c
	}

	// 2. IDで見つからない、または空の場合：IsPrimary キャラクターを探索
	var fallbackChar *domain.Character
	for _, c := range characters {
		if c.IsPrimary {
			return c // Primaryが見つかったら即座に採用
		}
		// 万が一 Primary が設定されていない場合のために、最初に見つかったキャラを控えておく
		if fallbackChar == nil {
			temp := c
			fallbackChar = &temp
		}
	}

	// 3. Primaryもいない場合：最初に見つかったキャラか、空のCharacterを返す
	if fallbackChar != nil {
		slog.Debug("Primary character not found, falling back to first available character", "name", fallbackChar.Name)
		return *fallbackChar
	}

	slog.Warn("No characters available in the map", "speakerID", page.SpeakerID)
	return domain.Character{Name: "Unknown"}
}
