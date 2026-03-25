package layout

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	imagePorts "github.com/shouni/gemini-image-kit/ports"
	"golang.org/x/sync/errgroup"

	"github.com/shouni/go-manga-kit/ports"
)

// negativePagePrompt は生成から除外したい要素を定義します。
const negativePagePrompt = "monochrome, black and white, greyscale, screentone, hatching, dot shades, ink sketch, line art only, realistic photos, 3d render, watermark, signature, deformed faces, bad anatomy, disfigured, poorly drawn hands, extra panels, unexpected panels, more than specified panels, split panels"

type PageGenerator struct {
	composer         *MangaComposer
	pb               ports.ImagePrompt
	maxPanelsPerPage int
}

// NewPageGenerator は、PageGeneratorの新しいインスタンスを作成します。
func NewPageGenerator(composer *MangaComposer, pb ports.ImagePrompt, maxPanelsPerPage int) *PageGenerator {
	return &PageGenerator{
		composer:         composer,
		pb:               pb,
		maxPanelsPerPage: maxPanelsPerPage,
	}
}

// Execute は、errgroupの制限機能を使用して並列数を制御しながらページ画像を生成します。
func (pg *PageGenerator) Execute(ctx context.Context, manga *ports.MangaResponse) ([]*imagePorts.ImageResponse, error) {
	if manga == nil || len(manga.Panels) == 0 {
		return nil, nil
	}

	if err := pg.composer.PrepareCharacterResources(ctx, manga.Panels); err != nil {
		return nil, fmt.Errorf("failed to prepare character resources: %w", err)
	}
	if err := pg.composer.PreparePanelResources(ctx, manga.Panels); err != nil {
		return nil, fmt.Errorf("failed to prepare panel resources: %w", err)
	}

	maxPanels := pg.maxPanelsPerPage
	if maxPanels <= 0 {
		maxPanels = defaultMaxPanelsPerPage
	}

	panelGroups := pg.chunkPanels(manga.Panels, maxPanels)
	totalPages := len(panelGroups)
	allResponses := make([]*imagePorts.ImageResponse, totalPages)

	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(int(pg.composer.MaxConcurrency))

	for i, group := range panelGroups {
		seed := pg.determineDefaultSeed(group)
		currentPageNum := i + 1

		eg.Go(func() error {
			if err := pg.composer.RateLimiter.Wait(egCtx); err != nil {
				return err
			}

			subManga := ports.MangaResponse{
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

			logger.Info("Manga page generation completed", "duration", time.Since(startTime).Round(time.Second))
			allResponses[i] = res
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return allResponses, nil
}

// generateMangaPage は、提供されたマンガレスポンスとAIベースの画像生成用のシードを使用して、マンガページの画像を生成します。
func (pg *PageGenerator) generateMangaPage(ctx context.Context, manga ports.MangaResponse, seed int64) (*imagePorts.ImageResponse, error) {
	// 1. リソース収集とインデックスマッピングの作成
	resMap := pg.collectResources(manga.Panels)

	// 2. プロンプト構築
	userPrompt, systemPrompt := pg.pb.BuildPage(manga.Panels, resMap)

	// 3. ImageURI 構造体のスライスを作成
	req := imagePorts.ImagePageRequest{
		GenerationOptions: imagePorts.GenerationOptions{
			Prompt:         userPrompt,
			SystemPrompt:   systemPrompt,
			NegativePrompt: negativePagePrompt,
			AspectRatio:    PageAspectRatio,
			ImageSize:      ImageSize2K,
			Seed:           &seed,
		},
		Images: resMap.OrderedAssets,
	}

	slog.Info("Requesting AI image generation",
		"title", manga.Title,
		"seed", seed,
		"total_assets", len(resMap.OrderedAssets),
	)

	return pg.composer.ImageGenerator.GenerateMangaPage(ctx, req)
}

// collectResources は、ページ内のキャラクター立ち絵とパネル参照画像を整理し、インデックスを割り振ります。
func (pg *PageGenerator) collectResources(panels []ports.Panel) *ports.ResourceMap {
	collector := newPageResourceCollector(pg.composer)
	collector.addCharacterAssets(panels)
	collector.addPanelAssets(panels)
	return collector.resourceMap
}

// chunkPanels はパネルのスライスを指定サイズのチャンクに分割して返します。
func (pg *PageGenerator) chunkPanels(panels []ports.Panel, size int) [][]ports.Panel {
	var chunks [][]ports.Panel
	for i := 0; i < len(panels); i += size {
		end := i + size
		if end > len(panels) {
			end = len(panels)
		}
		chunks = append(chunks, panels[i:end])
	}
	return chunks
}

// determineDefaultSeed はキャラクターデータを基にページ生成時のデフォルトシード値を決定します。
func (pg *PageGenerator) determineDefaultSeed(panels []ports.Panel) int64 {
	const defaultSeed = 1000
	if len(panels) == 0 {
		return defaultSeed
	}
	cm := pg.composer.CharactersMap

	// 最初のパネルの話者 Seed を優先します。
	if char := cm.GetCharacter(panels[0].SpeakerID); char != nil && char.Seed > 0 {
		return char.Seed
	}

	// 次にデフォルトキャラクターの Seed を参照します。
	if defaultChar := cm.GetDefault(); defaultChar != nil && defaultChar.Seed > 0 {
		return defaultChar.Seed
	}

	return defaultSeed
}
