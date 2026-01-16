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

// Execute は全ページを適切なチャンクに分割し、順次生成処理を実行します。
func (pg *PageGenerator) Execute(ctx context.Context, manga domain.MangaResponse) ([]*imagedom.ImageResponse, error) {
	var allResponses []*imagedom.ImageResponse

	if len(manga.Panels) == 0 {
		return allResponses, nil
	}

	// 1ページあたりの最大パネル数に基づいてチャンク分割
	totalPages := (len(manga.Panels) + MaxPanelsPerPage - 1) / MaxPanelsPerPage
	defaultSeed := pg.determineDefaultSeed(manga.Panels)
	var seedVal any = "none"
	if defaultSeed != nil {
		seedVal = *defaultSeed
	}

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
			Panels:      manga.Panels[i:end],
		}

		// 構造化ロギング
		logger := slog.With(
			"page_number", currentPageNum,
			"total_pages", totalPages,
			"panel_count", len(subManga.Panels),
			"seed", seedVal,
		)
		logger.Info("Starting manga page generation")

		res, err := pg.generateMangaPage(ctx, subManga, defaultSeed)
		if err != nil {
			return nil, fmt.Errorf("failed to generate page %d: %w", currentPageNum, err)
		}
		allResponses = append(allResponses, res)
	}

	return allResponses, nil
}

// generateMangaPage は構造化された台本を基に、1枚の統合漫画画像を生成します。
func (pg *PageGenerator) generateMangaPage(ctx context.Context, manga domain.MangaResponse, defaultSeed *int64) (*imagedom.ImageResponse, error) {
	pb := pg.mangaGenerator.PromptBuilder

	// 参照URLの収集
	refURLs := pg.collectReferences(manga.Panels)

	// プロンプト構築
	userPrompt, systemPrompt := pb.BuildMangaPagePrompt(manga.Panels, refURLs, manga.Title)
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

// determineDefaultSeed は、ページの代表的なSeed値を優先順位に基づいて決定します。
func (pg *PageGenerator) determineDefaultSeed(panels []domain.Panel) *int64 {
	// 1. PrimaryキャラクターのSeedを最優先で試みる
	if primaryChar := pg.mangaGenerator.Characters.GetPrimary(); primaryChar != nil && primaryChar.Seed > 0 {
		s := primaryChar.Seed
		return &s
	}

	// 2. Primaryが見つからない場合、登場順で最初の有効なSeedを持つキャラクターを探す
	for _, p := range panels {
		char := pg.mangaGenerator.Characters.FindCharacter(p.SpeakerID)
		if char != nil && char.Seed > 0 {
			s := char.Seed
			return &s
		}
	}

	return nil
}
