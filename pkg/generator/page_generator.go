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

func NewPageGenerator(composer *MangaComposer, pb prompts.ImagePrompt) *PageGenerator {
	return &PageGenerator{
		composer: composer,
		pb:       pb,
	}
}

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
	panelGroups := pg.chunkPanels(manga.Panels, MaxPanelsPerPage)
	totalPages := len(panelGroups)

	allResponses := make([]*imagedom.ImageResponse, totalPages)
	eg, egCtx := errgroup.WithContext(ctx)

	for i, group := range panelGroups {
		currentPageNum := i + 1
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
	// 1. リソース収集とインデックスマッピングの作成
	resMap, err := pg.collectResources(manga.Panels)
	if err != nil {
		return nil, fmt.Errorf("failed to collect resources: %w", err)
	}

	// 2. プロンプト構築 (ResourceMap を渡すように pb 側を調整済みと想定)
	userPrompt, systemPrompt := pg.pb.BuildMPage(manga.Panels, resMap)

	req := imagedom.ImagePageRequest{
		Prompt:         userPrompt,
		NegativePrompt: prompts.NegativePagePrompt,
		SystemPrompt:   systemPrompt,
		AspectRatio:    PageAspectRatio,
		Seed:           &seed,
		ReferenceURLs:  resMap.OrderedURLs,
		FileAPIURIs:    resMap.OrderedURIs,
	}

	slog.Info("Requesting AI image generation",
		"title", manga.Title,
		"seed", seed,
		"character_resources", len(resMap.CharacterFiles),
		"panel_resources", len(resMap.PanelFiles),
		"total_files", len(resMap.OrderedURIs),
	)

	return pg.composer.ImageGenerator.GenerateMangaPage(ctx, req)
}

// collectResources は、ページ内のキャラクター立ち絵とパネル参照画像を整理し、インデックスを割り振ります。
func (pg *PageGenerator) collectResources(panels []domain.Panel) (*prompts.ResourceMap, error) {
	res := &prompts.ResourceMap{
		CharacterFiles: make(map[string]int),
		PanelFiles:     make(map[string]int),
	}
	addedMap := make(map[string]int) // URL -> index (重複排除用)

	pg.composer.mu.RLock()
	defer pg.composer.mu.RUnlock()

	// 1. ページ内に登場する全キャラクターの立ち絵を優先的に登録 (input_file_0, 1...)
	speakerIDs := domain.Panels(panels).UniqueSpeakerIDs()
	for _, sID := range speakerIDs {
		char := pg.composer.CharactersMap.GetCharacter(sID)
		if char != nil && char.ReferenceURL != "" {
			if uri, ok := pg.composer.CharacterResourceMap[char.ID]; ok {
				// 同じURLが既に登録されていれば、そのインデックスを再利用
				if idx, exists := addedMap[char.ReferenceURL]; exists {
					res.CharacterFiles[sID] = idx
				} else {
					newIdx := len(res.OrderedURIs)
					res.OrderedURLs = append(res.OrderedURLs, char.ReferenceURL)
					res.OrderedURIs = append(res.OrderedURIs, uri)
					res.CharacterFiles[sID] = newIdx
					addedMap[char.ReferenceURL] = newIdx
				}
			}
		}
	}

	// 2. パネル固有のポーズ参照（重複排除しつつソートして順序を安定させる）
	type panelRef struct{ url, uri string }
	var refs []panelRef
	for _, p := range panels {
		if p.ReferenceURL == "" {
			continue
		}
		// 既にキャラ立ち絵として登録済みの画像でなければ、候補に追加
		if _, exists := addedMap[p.ReferenceURL]; !exists {
			if uri, ok := pg.composer.PanelResourceMap[p.ReferenceURL]; ok {
				refs = append(refs, panelRef{url: p.ReferenceURL, uri: uri})
				addedMap[p.ReferenceURL] = -1 // 仮登録
			}
		}
	}
	// 決定論的な順序のためにURLでソート
	sort.Slice(refs, func(i, j int) bool { return refs[i].url < refs[j].url })

	// 3. ソート済みパネル参照を OrderedURIs に追加し、インデックスを確定
	for _, r := range refs {
		newIdx := len(res.OrderedURIs)
		res.OrderedURLs = append(res.OrderedURLs, r.url)
		res.OrderedURIs = append(res.OrderedURIs, r.uri)
		res.PanelFiles[r.url] = newIdx
		// addedMap を更新（同じURLが複数パネルで使われている場合のため）
		addedMap[r.url] = newIdx
	}

	// 最後に、複数のパネルで同じ ReferenceURL が使われていた場合のマッピングを補完
	for _, p := range panels {
		if p.ReferenceURL != "" {
			if idx, ok := addedMap[p.ReferenceURL]; ok && idx != -1 {
				res.PanelFiles[p.ReferenceURL] = idx
			}
		}
	}

	return res, nil
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
