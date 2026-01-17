package generator

import (
	"context"
	"fmt"
	"log/slog"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/prompts"
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

// Execute は全パネルを適切なチャンク（ページ）に分割し、順次生成処理を実行します。
func (pg *PageGenerator) Execute(ctx context.Context, manga *domain.MangaResponse) ([]*imagedom.ImageResponse, error) {
	var allResponses []*imagedom.ImageResponse

	if len(manga.Panels) == 0 {
		return allResponses, nil
	}

	// 1ページあたりの最大パネル数に基づいてチャンク分割
	totalPages := (len(manga.Panels) + MaxPanelsPerPage - 1) / MaxPanelsPerPage
	seed := pg.determineDefaultSeed(manga.Panels)

	for i := 0; i < len(manga.Panels); i += MaxPanelsPerPage {
		if err := pg.composer.RateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter wait error: %w", err)
		}

		end := i + MaxPanelsPerPage
		if end > len(manga.Panels) {
			end = len(manga.Panels)
		}

		currentPageNum := (i / MaxPanelsPerPage) + 1

		// チャンク化されたページデータの作成
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
			"seed", seed,
		)
		logger.Info("Starting manga page generation")

		res, err := pg.generateMangaPage(ctx, subManga, seed)
		if err != nil {
			return nil, fmt.Errorf("failed to generate page %d: %w", currentPageNum, err)
		}
		allResponses = append(allResponses, res)
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

	// 画像生成リクエストの構築 (MangaComposer経由で呼び出し)
	req := imagedom.ImagePageRequest{
		Prompt:         userPrompt,
		NegativePrompt: prompts.NegativeMangaPagePrompt,
		SystemPrompt:   systemPrompt,
		AspectRatio:    PageAspectRatio,
		Seed:           &seed,
		ReferenceURLs:  refURLs,
	}

	return pg.composer.ImgGen.GenerateMangaPage(ctx, req)
}

// collectReferences は必要な全ての画像URLを重複なく収集します。
func (pg *PageGenerator) collectReferences(panels []domain.Panel) []string {
	urlMap := make(map[string]struct{})
	var urls []string
	cm := pg.composer.CharactersMap

	// デフォルトキャラクターの参照URLを最優先で常駐させる
	if def := cm.GetDefault(); def != nil && def.ReferenceURL != "" {
		urlMap[def.ReferenceURL] = struct{}{}
		urls = append(urls, def.ReferenceURL)
		slog.Debug("Default character reference added to page", "url", def.ReferenceURL)
	}

	for _, p := range panels {
		// パネル個別の参照画像
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
func (pg *PageGenerator) determineDefaultSeed(panels []domain.Panel) int64 {
	cm := pg.composer.CharactersMap

	// デフォルトキャラクターのSeedを最優先
	if defaultChar := cm.GetDefault(); defaultChar != nil && defaultChar.Seed > 0 {
		return defaultChar.Seed
	}

	// 登場順で最初の有効なSeedを持つキャラクターを探す
	for _, p := range panels {
		char := cm.GetCharacter(p.SpeakerID)
		if char != nil && char.Seed > 0 {
			return char.Seed
		}
	}

	// フォールバック
	// fallbackSeed は、キャラクター固有のSeedが見つからない場合に利用される固定のSeed値です。
	// この値自体に特別な意味はなく、生成結果の再現性を担保するためのデフォルト値として機能します。
	const fallbackSeed = 1000
	slog.Warn("キャラクター由来のシードが見つからないため、フォールバックシードを使用します",
		"fallback_seed", fallbackSeed,
		"panel_count", len(panels),
	)

	return fallbackSeed
}
