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
	"github.com/shouni/go-manga-kit/pkg/prompt"
)

// MaxPanelsPerPage は1枚の漫画ページに含めるパネルの最大数
const MaxPanelsPerPage = 6

// PageGenerator は複数のパネルを1枚の漫画ページとして統合生成する汎用部品なのだ。
type PageGenerator struct {
	mangaGenerator MangaGenerator
	styleSuffix    string
}

func NewPageGenerator(mangaGenerator MangaGenerator, styleSuffix string) *PageGenerator {
	return &PageGenerator{
		mangaGenerator: mangaGenerator,
		styleSuffix:    styleSuffix,
	}
}

// ExecuteMangaPages は複数ページをチャンクして生成するエントリーポイントなのだ
func (pg *PageGenerator) ExecuteMangaPages(ctx context.Context, manga domain.MangaResponse) ([]*imagedom.ImageResponse, error) {
	var allResponses []*imagedom.ImageResponse

	if len(manga.Pages) == 0 {
		slog.Warn("生成対象のページが空なのだ")
		return allResponses, nil
	}

	// 1ページあたりの最大パネル数に基づいて総ページ数を計算
	totalPages := (len(manga.Pages) + MaxPanelsPerPage - 1) / MaxPanelsPerPage

	slog.Info("一括生成パイプライン開始なのだ",
		slog.Int("total_panels", len(manga.Pages)),
		slog.Int("total_pages", totalPages),
	)

	for i := 0; i < len(manga.Pages); i += MaxPanelsPerPage {
		if i > 0 {
			waitSec := 30 // 500エラー対策で少し長めに設定するのだ
			slog.Info("APIの冷却期間なのだ。少し待機するのだよ...", slog.Int("wait_seconds", waitSec))

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(waitSec) * time.Second):
				// 待機完了
			}
		}

		// 2. チャンク範囲の決定
		end := i + MaxPanelsPerPage
		if end > len(manga.Pages) {
			end = len(manga.Pages)
		}

		currentPage := (i / MaxPanelsPerPage) + 1

		// 3. サブセット（そのページ専用のデータ）の作成
		subManga := domain.MangaResponse{
			Title:       fmt.Sprintf("%s (Page %d/%d)", manga.Title, currentPage, totalPages),
			Description: manga.Description,
			Pages:       manga.Pages[i:end],
		}

		slog.Info("ページの生成を試みるのだ",
			slog.Int("current_page", currentPage),
			slog.Int("total_pages", totalPages),
			slog.Int("panel_count", len(subManga.Pages)),
		)

		// 4. 単一ページの生成実行
		res, err := pg.ExecuteMangaPage(ctx, subManga)
		if err != nil {
			// どこで失敗したか明確にするのだ！
			slog.Error("ページ生成中にエラーが発生したのだ",
				slog.Int("page", currentPage),
				slog.Any("error", err),
			)
			return nil, fmt.Errorf("failed page %d: %w", currentPage-1, err)
		}

		allResponses = append(allResponses, res)
		slog.Info("ページの生成に成功したのだ！", slog.Int("page", currentPage))
	}

	slog.Info("全ページの生成が完了したのだ！ボクたちの完全勝利なのだよ！")
	return allResponses, nil
}

// ExecuteMangaPage は構造化された台本を基に、1枚の統合漫画画像を生成する
func (pg *PageGenerator) ExecuteMangaPage(ctx context.Context, manga domain.MangaResponse) (*imagedom.ImageResponse, error) {
	// 共通のスタイルサフィックス（anime styleなど）を注入して生成するのだ
	pb := prompt.NewPromptBuilder(pg.mangaGenerator.Characters, pg.styleSuffix)

	// 参照URLの収集
	refURLs := pg.collectReferences(manga.Pages, pg.mangaGenerator.Characters)

	// 巨大な統合プロンプトの構築
	fullPrompt := pb.BuildFullPagePrompt(manga.Title, manga.Pages, refURLs)

	// シード値の決定（最初のパネルのキャラを優先）
	var defaultSeed *int64
	if len(manga.Pages) > 0 {
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
func (pp *PageGenerator) collectReferences(pages []domain.MangaPage, characters map[string]domain.Character) []string {
	urlMap := make(map[string]struct{})
	var urls []string
	for _, p := range pages {
		if char := pp.findCharacter(p.SpeakerID, characters); char != nil && char.ReferenceURL != "" {
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
