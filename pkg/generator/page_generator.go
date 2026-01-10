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

// MaxPanelsPerPage は1枚の漫画ページに含めるパネルの最大数
const MaxPanelsPerPage = 6

// PageGenerator は複数のパネルを1枚の漫画ページとして統合生成する汎用部品なのだ。
type PageGenerator struct {
	mangaGenerator MangaGenerator
	styleSuffix    string
	limiter        *rate.Limiter
}

func NewPageGenerator(mangaGenerator MangaGenerator, styleSuffix string) *PageGenerator {
	return &PageGenerator{
		mangaGenerator: mangaGenerator,
		styleSuffix:    styleSuffix,
		limiter:        rate.NewLimiter(rate.Every(30*time.Second), 1),
	}
}

// ExecuteMangaPages は複数ページをチャンクして生成するエントリーポイントなのだ
func (pg *PageGenerator) ExecuteMangaPages(ctx context.Context, manga domain.MangaResponse) ([]*imagedom.ImageResponse, error) {
	var allResponses []*imagedom.ImageResponse

	if len(manga.Pages) == 0 {
		return allResponses, nil
	}

	totalPages := (len(manga.Pages) + MaxPanelsPerPage - 1) / MaxPanelsPerPage

	for i := 0; i < len(manga.Pages); i += MaxPanelsPerPage {
		slog.Info("APIリミッターのトークンを待機中なのだ...",
			slog.Int("current_idx", i),
		)
		if err := pg.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("リミッターの待機中にエラーが発生したのだ: %w", err)
		}

		// --- チャンク処理 ---
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

		slog.Info("生成リクエストを送信するのだ", slog.Int("page", currentPage))

		res, err := pg.ExecuteMangaPage(ctx, subManga)
		if err != nil {
			return nil, fmt.Errorf("failed page %d: %w", currentPage-1, err)
		}
		allResponses = append(allResponses, res)
	}

	return allResponses, nil
}

// ExecuteMangaPage は構造化された台本を基に、1枚の統合漫画画像を生成する
func (pg *PageGenerator) ExecuteMangaPage(ctx context.Context, manga domain.MangaResponse) (*imagedom.ImageResponse, error) {
	// 共通のスタイルサフィックス（anime styleなど）を注入して生成するのだ
	pb := prompts.NewPromptBuilder(pg.mangaGenerator.Characters, pg.styleSuffix)

	// 参照URLの収集
	refURLs := pg.collectReferences(manga.Pages, pg.mangaGenerator.Characters)

	// 巨大な統合プロンプトの構築
	fullPrompt := pb.BuildFullPagePrompt(manga.Title, manga.Pages, refURLs)

	var defaultSeed *int64

	// 1. まずはページ内の全パネルを走査して、IsPrimaryなキャラ（二人組など）がいるか確認するのだ
	for _, p := range manga.Pages {
		char := pg.findCharacter(p.SpeakerID, pg.mangaGenerator.Characters)
		if char != nil && char.IsPrimary && char.Seed > 0 {
			s := char.Seed
			defaultSeed = &s

			break // 最優先キャラが見つかったら、即座に採用してループを抜けるのだ
		}
	}

	// 2. もしIsPrimaryなキャラがいなかったら、従来通り最初のパネルの話者のSeedを使うのだ
	if defaultSeed == nil && len(manga.Pages) > 0 {
		char := pg.findCharacter(manga.Pages[0].SpeakerID, pg.mangaGenerator.Characters)
		if char != nil && char.Seed > 0 {
			s := char.Seed
			defaultSeed = &s
		}
	}

	req := imagedom.ImagePageRequest{
		Prompt:         fullPrompt,
		NegativePrompt: "deformed faces, mismatched eyes, cross-eyed, low-quality faces, blurry facial features, melting faces, extra limbs, merged panels, messy lineart, distorted anatomy",
		AspectRatio:    "3:4",
		Seed:           defaultSeed,
		ReferenceURLs:  refURLs,
	}

	return pg.mangaGenerator.ImgGen.GenerateMangaPage(ctx, req)
}

// findCharacter は SpeakerID（名前またはハッシュ化ID）からキャラを特定するのだ
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

// collectReferences は必要な全ての画像URLを重複なく収集するのだ
func (pg *PageGenerator) collectReferences(pages []domain.MangaPage, characters map[string]domain.Character) []string {
	urlMap := make(map[string]struct{})
	var urls []string
	for _, p := range pages {
		if char := pg.findCharacter(p.SpeakerID, characters); char != nil && char.ReferenceURL != "" {
			if _, exists := urlMap[char.ReferenceURL]; !exists {
				urlMap[char.ReferenceURL] = struct{}{}
				urls = append(urls, char.ReferenceURL)
			}
		}
	}
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
