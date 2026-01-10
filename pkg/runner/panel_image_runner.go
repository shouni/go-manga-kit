package runner

import (
	"context"
	"log/slog"
	"time"

	"github.com/shouni/go-manga-kit/pkg/workflow"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/generator"
)

// MangaPanelImageRunner は、台本を元に並列画像生成を管理する実装なのだ。
type MangaPanelImageRunner struct {
	cfg      workflow.Config
	groupGen *generator.GroupGenerator // 並列生成とレートリミットを管理するコアなのだ
}

// NewMangaPanelImageRunner は、依存関係を注入して初期化するのだ。
// config.DefaultRateLimit などの具体的な値は builder から渡されることを想定しているのだ。
func NewMangaPanelImageRunner(cfg workflow.Config, mangaGen generator.MangaGenerator, styleSuffix string, interval time.Duration) *MangaPanelImageRunner {
	groupGen := generator.NewGroupGenerator(mangaGen, styleSuffix, interval)

	return &MangaPanelImageRunner{
		cfg:      cfg,
		groupGen: groupGen,
	}
}

// Run は、台本(MangaResponse)を受け取り、指定されたパネルの画像を生成するのだ。
func (r *MangaPanelImageRunner) Run(ctx context.Context, manga domain.MangaResponse, targetIndices []int) ([]*imagedom.ImageResponse, error) {
	allPages := manga.Pages
	var targetPages []domain.MangaPage

	// 1. 生成対象のフィルタリング
	if len(targetIndices) > 0 {
		slog.Info("Generating specific panels", "indices", targetIndices)
		for _, idx := range targetIndices {
			if idx >= 0 && idx < len(allPages) {
				targetPages = append(targetPages, allPages[idx])
			} else {
				slog.Warn("Index out of range, skipping", "index", idx, "total_pages", len(allPages))
			}
		}
	} else {
		// 指定がない場合は全件対象なのだ
		targetPages = allPages
	}

	if len(targetPages) == 0 {
		slog.Info("No pages to generate.")
		return []*imagedom.ImageResponse{}, nil
	}

	slog.Info("Starting parallel image generation",
		"title", manga.Title,
		"target_count", len(targetPages),
		"total_count", len(allPages),
	)

	// 2. 既存の manga-kit のロジック（GroupGenerator）に委譲するのだ
	images, err := r.groupGen.ExecutePanelGroup(ctx, targetPages)
	if err != nil {
		slog.Error("Image generation pipeline failed", "error", err)
		return nil, err
	}

	slog.Info("Successfully generated panels", "count", len(images))
	return images, nil
}
