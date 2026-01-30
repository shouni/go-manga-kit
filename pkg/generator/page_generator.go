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
	pb       prompts.ImagePrompt
}

// NewPageGenerator は、PageGeneratorの新しいインスタンスを作成します。
func NewPageGenerator(composer *MangaComposer, pb prompts.ImagePrompt) *PageGenerator {
	return &PageGenerator{
		composer: composer,
		pb:       pb,
	}
}

// Execute は、そのページの画像レスポンスを並行して生成します。
func (pg *PageGenerator) Execute(ctx context.Context, manga *domain.MangaResponse) ([]*imagedom.ImageResponse, error) {
	if len(manga.Panels) == 0 {
		return nil, nil
	}

	// 1. アセットの事前並列アップロード（Character と Panel のリソースを準備）
	if err := pg.composer.PrepareCharacterResources(ctx, manga.Panels); err != nil {
		return nil, fmt.Errorf("failed to prepare character resources: %w", err)
	}
	if err := pg.composer.PreparePanelResources(ctx, manga.Panels); err != nil {
		return nil, fmt.Errorf("failed to prepare panel resources: %w", err)
	}

	// 2. ページ分割と並列実行の準備
	maxPanels := pg.composer.Config.MaxPanelsPerPage
	if maxPanels <= 0 {
		maxPanels = DefaultMaxPanelsPerPage
	}

	panelGroups := pg.chunkPanels(manga.Panels, maxPanels)
	totalPages := len(panelGroups)

	allResponses := make([]*imagedom.ImageResponse, totalPages)
	eg, egCtx := errgroup.WithContext(ctx)

	for i, group := range panelGroups {
		seed := pg.determineDefaultSeed(group)
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

// generateMangaPage は、提供されたマンガレスポンスとAIベースの画像生成用のシードを使用して、マンガページの画像を生成します。
func (pg *PageGenerator) generateMangaPage(ctx context.Context, manga domain.MangaResponse, seed int64) (*imagedom.ImageResponse, error) {
	// 1. リソース収集とインデックスマッピングの作成
	resMap, err := pg.collectResources(manga.Panels)
	if err != nil {
		return nil, fmt.Errorf("failed to collect resources: %w", err)
	}

	// 2. プロンプト構築
	userPrompt, systemPrompt := pg.pb.BuildPage(manga.Panels, resMap)

	// 3. ImageURI 構造体のスライスを作成
	req := imagedom.ImagePageRequest{
		Prompt:         userPrompt,
		NegativePrompt: prompts.NegativePagePrompt,
		SystemPrompt:   systemPrompt,
		AspectRatio:    PageAspectRatio,
		ImageSize:      ImageSize2K,
		Images:         resMap.OrderedAssets,
		Seed:           &seed,
	}

	slog.Info("Requesting AI image generation",
		"title", manga.Title,
		"seed", seed,
		"total_assets", len(resMap.OrderedAssets),
	)

	return pg.composer.ImageGenerator.GenerateMangaPage(ctx, req)
}

// collectResources は、ページ内のキャラクター立ち絵とパネル参照画像を整理し、インデックスを割り振ります。
func (pg *PageGenerator) collectResources(panels []domain.Panel) (*prompts.ResourceMap, error) {
	res := &prompts.ResourceMap{
		CharacterFiles: make(map[string]int),
		PanelFiles:     make(map[string]int),
		OrderedAssets:  []imagedom.ImageURI{},
	}
	addedMap := make(map[string]int) // URL -> index (重複排除用)

	pg.composer.mu.RLock()
	defer pg.composer.mu.RUnlock()

	// 1. ページ内に登場する全キャラクターの立ち絵を優先的に登録
	speakerIDs := domain.Panels(panels).UniqueSpeakerIDs()
	for _, sID := range speakerIDs {
		char := pg.composer.CharactersMap.GetCharacter(sID)
		if char != nil && char.ReferenceURL != "" {
			if uri, ok := pg.composer.CharacterResourceMap[char.ID]; ok {
				if idx, exists := addedMap[char.ReferenceURL]; exists {
					res.CharacterFiles[sID] = idx
				} else {
					newIdx := len(res.OrderedAssets)
					res.OrderedAssets = append(res.OrderedAssets, imagedom.ImageURI{
						ReferenceURL: char.ReferenceURL,
						FileAPIURI:   uri,
					})
					res.CharacterFiles[sID] = newIdx
					addedMap[char.ReferenceURL] = newIdx
				}
			}
		}
	}

	// 2. パネル固有のポーズ参照
	var panelRefs []imagedom.ImageURI
	for _, p := range panels {
		if p.ReferenceURL == "" {
			continue
		}
		if _, exists := addedMap[p.ReferenceURL]; !exists {
			if uri, ok := pg.composer.PanelResourceMap[p.ReferenceURL]; ok {
				panelRefs = append(panelRefs, imagedom.ImageURI{
					ReferenceURL: p.ReferenceURL,
					FileAPIURI:   uri,
				})
				addedMap[p.ReferenceURL] = -1 // 仮登録
			}
		}
	}

	// 決定論的な順序のためにURLでソート
	sort.Slice(panelRefs, func(i, j int) bool {
		return panelRefs[i].ReferenceURL < panelRefs[j].ReferenceURL
	})

	// 3. ソート済みパネル参照を OrderedAssets に追加し、インデックスを確定
	for _, r := range panelRefs {
		newIdx := len(res.OrderedAssets)
		res.OrderedAssets = append(res.OrderedAssets, r)
		res.PanelFiles[r.ReferenceURL] = newIdx
		addedMap[r.ReferenceURL] = newIdx
	}

	// 4. 重複していた ReferenceURL のマッピングを補完
	for _, p := range panels {
		if p.ReferenceURL != "" {
			if idx, ok := addedMap[p.ReferenceURL]; ok && idx != -1 {
				res.PanelFiles[p.ReferenceURL] = idx
			}
		}
	}

	return res, nil
}

// chunkPanels パネルのスライスを指定されたサイズのチャンクに分割し、チャンクの 2D スライスを返します。
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

// determineDefaultSeed 利用可能なキャラクターデータに基づいて、マンガパネル生成のデフォルトシード値を決定します
func (pg *PageGenerator) determineDefaultSeed(panels []domain.Panel) int64 {
	const defaultSeed = 1000
	if len(panels) == 0 {
		return defaultSeed
	}
	cm := pg.composer.CharactersMap

	// パネルの最初のキャラのseed
	if char := cm.GetCharacter(panels[0].SpeakerID); char != nil && char.Seed > 0 {
		return char.Seed
	}

	// デフォルトのキャラseed
	if defaultChar := cm.GetDefault(); defaultChar != nil && defaultChar.Seed > 0 {
		return defaultChar.Seed
	}

	return defaultSeed
}
