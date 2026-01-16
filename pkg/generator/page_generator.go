package generator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/prompts"
	"golang.org/x/time/rate"
)

// PageGenerator は複数のパネルを1枚の漫画ページとして統合生成するコンポーネントです。
type PageGenerator struct {
	mangaGenerator MangaGenerator
	limiter        *rate.Limiter
}

// NewPageGenerator は PageGenerator の新しいインスタンスを初期化します。
func NewPageGenerator(mangaGenerator MangaGenerator, interval time.Duration) *PageGenerator {
	return &PageGenerator{
		mangaGenerator: mangaGenerator,
		limiter:        rate.NewLimiter(rate.Every(interval), 1),
	}
}

// ExecuteMangaPages は全ページを適切なチャンクに分割し、順次生成処理を実行します。
func (pg *PageGenerator) ExecuteMangaPages(ctx context.Context, manga domain.MangaResponse) ([]*imagedom.ImageResponse, error) {
	var allResponses []*imagedom.ImageResponse

	if len(manga.Panels) == 0 {
		return allResponses, nil
	}

	// 1ページあたりの最大パネル数に基づいてチャンク分割
	totalPages := (len(manga.Panels) + MaxPanelsPerPage - 1) / MaxPanelsPerPage

	for i := 0; i < len(manga.Panels); i += MaxPanelsPerPage {
		if err := pg.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter wait error: %w", err)
		}

		end := i + MaxPanelsPerPage
		if end > len(manga.Panels) {
			end = len(manga.Panels)
		}

		currentPageNum := (i / MaxPanelsPerPage) + 1

		// チャンク化された台本データの作成
		subManga := domain.MangaResponse{
			Title:       fmt.Sprintf("%s (Page %d/%d)", manga.Title, currentPageNum, totalPages),
			Description: manga.Description,
			Panels:      manga.Panels[i:end], // フィールド名を Pages に統一
		}

		// 構造化ロギング
		logger := slog.With(
			"page_number", currentPageNum,
			"total_pages", totalPages,
			"panel_count", len(subManga.Panels),
		)
		logger.Info("Starting manga page generation")

		res, err := pg.ExecuteMangaPage(ctx, subManga)
		if err != nil {
			return nil, fmt.Errorf("failed to generate page %d: %w", currentPageNum, err)
		}
		allResponses = append(allResponses, res)
	}

	return allResponses, nil
}

// ExecuteMangaPage は構造化された台本を基に、1枚の統合漫画画像を生成します。
func (pg *PageGenerator) ExecuteMangaPage(ctx context.Context, manga domain.MangaResponse) (*imagedom.ImageResponse, error) {
	pb := pg.mangaGenerator.PromptBuilder

	// 参照URLの収集
	refURLs := pg.collectReferences(manga.Panels)

	// プロンプト構築 (User/System プロンプトの分離)
	userPrompt, systemPrompt := pb.BuildMangaPagePrompt(manga.Title, manga.Panels, refURLs)

	// キャラクター設定からSeed値を特定
	var defaultSeed *int64
	for _, p := range manga.Panels {
		char := pg.mangaGenerator.Characters.FindCharacter(p.SpeakerID)
		if char != nil && char.Seed > 0 {
			// Primary キャラクターがいれば最優先で採用
			if char.IsPrimary {
				s := char.Seed
				defaultSeed = &s
				break
			}
			// Primary が見つからない場合の暫定フォールバック
			if defaultSeed == nil {
				s := char.Seed
				defaultSeed = &s
			}
		}
	}

	// 画像生成リクエストの構築
	req := imagedom.ImagePageRequest{
		Prompt:         userPrompt,
		NegativePrompt: prompts.DefaultNegativeMangaPagePrompt,
		SystemPrompt:   systemPrompt,
		AspectRatio:    PageAspectRatio,
		Seed:           defaultSeed,
		ReferenceURLs:  refURLs,
	}

	return pg.mangaGenerator.ImgGen.GenerateMangaPage(ctx, req)
}

// collectReferences は必要な全ての画像URLを重複なく収集します。
func (pg *PageGenerator) collectReferences(pages []domain.Panel) []string {
	urlMap := make(map[string]struct{})
	var urls []string

	for _, p := range pages {
		char := pg.mangaGenerator.Characters.FindCharacter(p.SpeakerID)

		// 1. キャラクターのマスター参照画像
		if char != nil && char.ReferenceURL != "" {
			if _, exists := urlMap[char.ReferenceURL]; !exists {
				urlMap[char.ReferenceURL] = struct{}{}
				urls = append(urls, char.ReferenceURL)
			}
		}

		// 2. パネル個別の参照画像
		if p.ReferenceURL != "" {
			if _, exists := urlMap[p.ReferenceURL]; !exists {
				urlMap[p.ReferenceURL] = struct{}{}
				urls = append(urls, p.ReferenceURL)
			}
		}
	}
	return urls
}
