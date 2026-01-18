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
)

// PageGenerator は複数のパネルを1枚の漫画ページとして統合生成するコンポーネントです。
type PageGenerator struct {
	composer *MangaComposer
}

// NewPageGenerator は PageGenerator の新しいインスタンスを初期化します。
func NewPageGenerator(composer *MangaComposer) *PageGenerator {
	return &PageGenerator{
		composer: composer,
	}
}

// Execute は全パネルを適切なページに分割し、並列で生成処理を実行します。
func (pg *PageGenerator) Execute(ctx context.Context, manga *domain.MangaResponse) ([]*imagedom.ImageResponse, error) {
	if len(manga.Panels) == 0 {
		return nil, nil
	}

	panelGroups := pg.chunkPanels(manga.Panels, MaxPanelsPerPage)
	totalPages := len(panelGroups)
	seed := pg.determineDefaultSeed(manga.Panels)

	allResponses := make([]*imagedom.ImageResponse, totalPages)
	eg, egCtx := errgroup.WithContext(ctx)

	for i, group := range panelGroups {
		currentPageNum := i + 1

		eg.Go(func() error {
			if err := pg.composer.RateLimiter.Wait(egCtx); err != nil {
				return fmt.Errorf("rate limiter wait error: %w", err)
			}

			subManga := domain.MangaResponse{
				Title:       fmt.Sprintf("%s (Page %d/%d)", manga.Title, currentPageNum, totalPages),
				Description: manga.Description,
				Panels:      group,
			}

			logger := slog.With(
				"page_number", currentPageNum,
				"total_pages", totalPages,
				"panel_count", len(group),
				"seed", seed,
			)
			logger.Info("Starting manga page generation")

			startTime := time.Now()
			res, err := pg.generateMangaPage(egCtx, subManga, seed)
			if err != nil {
				return fmt.Errorf("failed to generate page %d: %w", currentPageNum, err)
			}

			logger.Info("Manga page generation completed", "duration", time.Since(startTime).Round(time.Millisecond))
			allResponses[i] = res
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return allResponses, nil
}

// generateMangaPage は構造化された情報を基に、1枚の統合漫画画像を生成します。
func (pg *PageGenerator) generateMangaPage(ctx context.Context, manga domain.MangaResponse, seed int64) (*imagedom.ImageResponse, error) {
	pb := pg.composer.PromptBuilder

	// 参照URLの収集
	refURLs := pg.collectReferences(manga.Panels)

	// プロンプト構築
	userPrompt, systemPrompt := pb.BuildMangaPagePrompt(manga.Panels, refURLs)

	// 画像生成リクエストの構築
	req := imagedom.ImagePageRequest{
		Prompt:         userPrompt,
		NegativePrompt: prompts.NegativeMangaPagePrompt,
		SystemPrompt:   systemPrompt,
		AspectRatio:    PageAspectRatio,
		Seed:           &seed,
		ReferenceURLs:  refURLs,
	}

	return pg.composer.ImageGenerator.GenerateMangaPage(ctx, req)
}

// collectReferences は必要な全ての画像URLを重複なく収集します。
// 出力順序を決定論的にするため、ソート処理を追加しています。
func (pg *PageGenerator) collectReferences(panels []domain.Panel) []string {
	urlMap := make(map[string]struct{})
	cm := pg.composer.CharactersMap

	// デフォルトキャラクターの参照URL
	if def := cm.GetDefault(); def != nil && def.ReferenceURL != "" {
		urlMap[def.ReferenceURL] = struct{}{}
	}

	// 各パネルの参照画像
	for _, p := range panels {
		if p.ReferenceURL != "" {
			urlMap[p.ReferenceURL] = struct{}{}
		}
	}

	// 順序を一定にするためにソート
	urls := make([]string, 0, len(urlMap))
	for url := range urlMap {
		urls = append(urls, url)
	}
	sort.Strings(urls)

	return urls
}

// chunkPanels はパネルのスライスを指定されたサイズごとに分割します。
func (pg *PageGenerator) chunkPanels(panels []domain.Panel, size int) [][]domain.Panel {
	var chunks [][]domain.Panel
	for i := 0; i < len(panels); i += size {
		end := i + size
		if end > len(panels) {
			end = len(panels)
		}
		chunks = append(chunks, panels[i:end])
	}
	return chunks
}

// determineDefaultSeed は、ページの代表的なSeed値を優先順位に基づいて決定します。
func (pg *PageGenerator) determineDefaultSeed(panels []domain.Panel) int64 {
	cm := pg.composer.CharactersMap

	// 1. デフォルトキャラクターのSeedを最優先
	if defaultChar := cm.GetDefault(); defaultChar != nil && defaultChar.Seed > 0 {
		return defaultChar.Seed
	}

	// 2. 登場するキャラクターから有効なSeedを検索
	for _, p := range panels {
		char := cm.GetCharacter(p.SpeakerID)
		if char != nil && char.Seed > 0 {
			return char.Seed
		}
	}

	// 3. フォールバック（警告ログを復元）
	const fallbackSeed = 1000
	slog.Warn("No character-specific seed found, using fallback seed. This may affect visual consistency.",
		"fallback_seed", fallbackSeed,
		"panel_count", len(panels),
	)

	return fallbackSeed
}
