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

	// アセットの事前並列アップロード
	if err := pg.composer.PrepareCharacterResources(ctx, manga.Panels); err != nil {
		return nil, err
	}
	if err := pg.composer.PreparePanelResources(ctx, manga.Panels); err != nil {
		return nil, err
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

			logger := slog.With("page", currentPageNum, "total", totalPages, "seed", seed)
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

	// 生のURL(プロンプト用)と File API URI(画像データ用)を分離して収集
	rawURLs, fileURIs := pg.collectResources(manga.Panels)

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

	return pg.composer.ImageGenerator.GenerateMangaPage(ctx, req)
}

func (pg *PageGenerator) collectResources(panels []domain.Panel) (rawURLs []string, fileURIs []string) {
	rawMap := make(map[string]struct{})
	fileMap := make(map[string]struct{})
	cm := pg.composer.CharactersMap

	add := func(raw, file string) {
		if raw != "" {
			if _, ok := rawMap[raw]; !ok {
				rawMap[raw] = struct{}{}
				rawURLs = append(rawURLs, raw)
			}
		}
		if file != "" {
			if _, ok := fileMap[file]; !ok {
				fileMap[file] = struct{}{}
				fileURIs = append(fileURIs, file)
			}
		}
	}

	charIDsToCollect := make(map[string]string) // charID -> referenceURL
	panelURLsToCollect := make(map[string]struct{})

	// 1. 収集フェーズ (ロックなし)
	if def := cm.GetDefault(); def != nil {
		charIDsToCollect[def.ID] = def.ReferenceURL
	}
	for _, p := range panels {
		if char := cm.GetCharacter(p.SpeakerID); char != nil {
			charIDsToCollect[char.ID] = char.ReferenceURL
		}
		if p.ReferenceURL != "" {
			panelURLsToCollect[p.ReferenceURL] = struct{}{}
		}
	}

	// 2. マップアクセスフェーズ (ロックあり)
	// ロック時間を最小化するため、I/Oや計算を含まない抽出のみを行う
	pg.composer.mu.RLock()
	for id, refURL := range charIDsToCollect {
		add(refURL, pg.composer.CharacterResourceMap[id])
	}
	for refURL := range panelURLsToCollect {
		add(refURL, pg.composer.PanelResourceMap[refURL])
	}
	pg.composer.mu.RUnlock()

	// 3. ソートフェーズ (ロックなし)
	sort.Strings(rawURLs)
	sort.Strings(fileURIs)
	return rawURLs, fileURIs
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

	if defaultChar := cm.GetDefault(); defaultChar != nil && defaultChar.Seed > 0 {
		return defaultChar.Seed
	}

	for _, p := range panels {
		char := cm.GetCharacter(p.SpeakerID)
		if char != nil && char.Seed > 0 {
			return char.Seed
		}
	}

	const fallbackSeed = 1000
	slog.Warn("No character-specific seed found, using fallback seed. This may affect visual consistency.",
		"fallback_seed", fallbackSeed,
		"panel_count", len(panels),
	)

	return fallbackSeed
}
