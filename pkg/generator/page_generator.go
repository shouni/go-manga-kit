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

type PageGenerator struct {
	composer *MangaComposer
}

func NewPageGenerator(composer *MangaComposer) *PageGenerator {
	return &PageGenerator{composer: composer}
}

func (pg *PageGenerator) Execute(ctx context.Context, manga *domain.MangaResponse) ([]*imagedom.ImageResponse, error) {
	if len(manga.Panels) == 0 {
		return nil, nil
	}

	// 1. アセットの事前並列アップロード（これらが成功していることが後続の前提条件）
	if err := pg.composer.PrepareCharacterResources(ctx, manga.Panels); err != nil {
		return nil, fmt.Errorf("failed to prepare character resources: %w", err)
	}
	if err := pg.composer.PreparePanelResources(ctx, manga.Panels); err != nil {
		return nil, fmt.Errorf("failed to prepare panel resources: %w", err)
	}

	// 2. ページ分割と並列実行の準備
	panelGroups := pg.chunkPanels(manga.Panels, MaxPanelsPerPage)
	totalPages := len(panelGroups)

	allResponses := make([]*imagedom.ImageResponse, totalPages)
	eg, egCtx := errgroup.WithContext(ctx)

	for i, group := range panelGroups {
		currentPageNum := i + 1
		// ページ冒頭のキャラクターDNAを引き継ぐためのSeed決定
		seed := pg.determineDefaultSeed(group)

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
				"page", currentPageNum,
				"total", totalPages,
				"panels", len(group),
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

func (pg *PageGenerator) generateMangaPage(ctx context.Context, manga domain.MangaResponse, seed int64) (*imagedom.ImageResponse, error) {
	pb := pg.composer.PromptBuilder

	// リソース収集：パネル固有のReferenceURLをソート済みリストとして取得
	rawURLs, fileURIs, err := pg.collectResources(manga.Panels)
	if err != nil {
		return nil, fmt.Errorf("failed to collect resources: %w", err)
	}

	userPrompt, systemPrompt := pb.BuildMangaPagePrompt(manga.Panels, rawURLs)

	req := imagedom.ImagePageRequest{
		Prompt:         userPrompt,
		NegativePrompt: prompts.NegativeMangaPagePrompt,
		SystemPrompt:   systemPrompt,
		AspectRatio:    PageAspectRatio,
		Seed:           &seed,
		ReferenceURLs:  rawURLs,
		FileAPIURIs:    fileURIs,
	}

	slog.Info("Requesting AI image generation",
		"title", manga.Title,
		"seed", seed,
		"use_file_api", len(fileURIs),
	)

	return pg.composer.ImageGenerator.GenerateMangaPage(ctx, req)
}

// collectResources は、ページ内のパネルが要求する外部リソースを
// 重複なく、かつURL順で決定論的に収集します。
func (pg *PageGenerator) collectResources(panels []domain.Panel) (rawURLs []string, fileURIs []string, err error) {
	type resourceEntry struct {
		rawURL  string
		fileURI string
	}
	var entries []resourceEntry
	addedMap := make(map[string]struct{})

	pg.composer.mu.RLock()
	defer pg.composer.mu.RUnlock()

	for _, p := range panels {
		if p.ReferenceURL == "" {
			continue
		}
		if _, ok := addedMap[p.ReferenceURL]; !ok {
			// パネルリソースマップからURIを取得
			if uri, ok := pg.composer.PanelResourceMap[p.ReferenceURL]; ok {
				entries = append(entries, resourceEntry{rawURL: p.ReferenceURL, fileURI: uri})
				addedMap[p.ReferenceURL] = struct{}{}
			}
		}
	}

	// input_file_N の対応関係を一定にするためソート
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].rawURL < entries[j].rawURL
	})

	for _, entry := range entries {
		rawURLs = append(rawURLs, entry.rawURL)
		fileURIs = append(fileURIs, entry.fileURI)
	}

	return rawURLs, fileURIs, nil
}

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

// determineDefaultSeed は、ページの最初のパネルのキャラクターSeedを最優先し、絵柄を安定させます。
func (pg *PageGenerator) determineDefaultSeed(panels []domain.Panel) int64 {
	if len(panels) == 0 {
		return 1000
	}

	cm := pg.composer.CharactersMap

	// 1. 最初のパネルのスピーカーのSeedを優先
	if char := cm.GetCharacter(panels[0].SpeakerID); char != nil && char.Seed > 0 {
		return char.Seed
	}

	// 2. フォールバック: システム全体のデフォルトキャラSeed
	if defaultChar := cm.GetDefault(); defaultChar != nil && defaultChar.Seed > 0 {
		return defaultChar.Seed
	}

	return 1000
}
