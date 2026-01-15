package generator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
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
func NewPageGenerator(mangaGenerator MangaGenerator) *PageGenerator {
	return &PageGenerator{
		mangaGenerator: mangaGenerator,
		limiter:        rate.NewLimiter(rate.Every(30*time.Second), 1),
	}
}

// ExecuteMangaPages は全ページを適切なチャンクに分割し、順次生成処理を実行します。
func (pg *PageGenerator) ExecuteMangaPages(ctx context.Context, manga domain.MangaResponse) ([]*imagedom.ImageResponse, error) {
	var allResponses []*imagedom.ImageResponse

	if len(manga.Pages) == 0 {
		return allResponses, nil
	}

	totalPages := (len(manga.Pages) + MaxPanelsPerPage - 1) / MaxPanelsPerPage

	for i := 0; i < len(manga.Pages); i += MaxPanelsPerPage {
		slog.Info("APIレート制限を確認中...", slog.Int("current_idx", i))

		if err := pg.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("リミッター待機中にエラーが発生しました: %w", err)
		}

		end := i + MaxPanelsPerPage
		if end > len(manga.Pages) {
			end = len(manga.Pages)
		}

		currentPage := (i / MaxPanelsPerPage) + 1
		subManga := domain.MangaResponse{
			Title:       fmt.Sprintf("%s (Page %d/%d)", manga.Title, currentPage, totalPages),
			Description: manga.Description,
			Pages:       manga.Pages[i:end],
		}

		slog.Info("ページ生成リクエストを送信します", slog.Int("page", currentPage))

		res, err := pg.ExecuteMangaPage(ctx, subManga)
		if err != nil {
			return nil, fmt.Errorf("ページ %d の生成に失敗しました: %w", currentPage, err)
		}
		allResponses = append(allResponses, res)
	}

	return allResponses, nil
}

// ExecuteMangaPage は構造化された台本を基に、1枚の統合漫画画像を生成します。
func (pg *PageGenerator) ExecuteMangaPage(ctx context.Context, manga domain.MangaResponse) (*imagedom.ImageResponse, error) {
	// 共通のスタイルサフィックスを使用してプロンプトビルダーを初期化
	pb := pg.mangaGenerator.PromptBuilder

	// 参照URLの収集
	refURLs := pg.collectReferences(manga.Pages, pg.mangaGenerator.Characters)

	// ページ全体のプロンプトを構築
	userPrompt, systemPrompt := pb.BuildMangaPagePrompt(manga.Title, manga.Pages, refURLs)

	var defaultSeed *int64

	// 優先度の高いキャラクターのSeed値を探索
	for _, p := range manga.Pages {
		char := pg.findCharacter(p.SpeakerID, pg.mangaGenerator.Characters)
		if char != nil && char.IsPrimary && char.Seed > 0 {
			s := char.Seed
			defaultSeed = &s
			break
		}
	}

	// 優先キャラがいない場合、最初の話者のSeedをフォールバックとして使用
	if defaultSeed == nil && len(manga.Pages) > 0 {
		char := pg.findCharacter(manga.Pages[0].SpeakerID, pg.mangaGenerator.Characters)
		if char != nil && char.Seed > 0 {
			s := char.Seed
			defaultSeed = &s
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

// findCharacter は SpeakerID からキャラクター情報を特定します。
func (pg *PageGenerator) findCharacter(speakerID string, characters map[string]domain.Character) *domain.Character {
	sid := strings.ToLower(speakerID)
	h := sha256.New()
	for _, char := range characters {
		h.Reset()
		h.Write([]byte(char.ID))
		hash := hex.EncodeToString(h.Sum(nil))
		if sid == "speaker-"+hash[:10] {
			return &char
		}
	}
	cleanID := strings.TrimPrefix(sid, "speaker-")
	if char, ok := characters[cleanID]; ok {
		return &char
	}
	return nil
}

// collectReferences は必要な全ての画像URLを重複なく収集します。
func (pg *PageGenerator) collectReferences(pages []domain.MangaPage, characters map[string]domain.Character) []string {
	urlMap := make(map[string]struct{})
	var urls []string

	// キャラクターの参照URL
	for _, p := range pages {
		if char := pg.findCharacter(p.SpeakerID, characters); char != nil && char.ReferenceURL != "" {
			if _, exists := urlMap[char.ReferenceURL]; !exists {
				urlMap[char.ReferenceURL] = struct{}{}
				urls = append(urls, char.ReferenceURL)
			}
		}
	}

	// ページごとの個別参照URL
	for _, p := range pages {
		if p.ReferenceURL != "" {
			if _, exists := urlMap[p.ReferenceURL]; !exists {
				urlMap[p.ReferenceURL] = struct{}{}
				urls = append(urls, p.ReferenceURL)
			}
		}
	}
	return urls
}
