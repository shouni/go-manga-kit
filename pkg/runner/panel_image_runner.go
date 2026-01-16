package runner

import (
	"context"
	"log/slog"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

// MangaPanelImageRunner は、台本を元に並列画像生成を管理します。
type MangaPanelImageRunner struct {
	cfg       config.Config
	generator generator.PanelsImageGenerator
	writer    remoteio.OutputWriter
}

// NewMangaPanelImageRunner は、依存関係を注入して初期化します。
func NewMangaPanelImageRunner(
	cfg config.Config,
	generator generator.PanelsImageGenerator,
	writer remoteio.OutputWriter,
) *MangaPanelImageRunner {
	return &MangaPanelImageRunner{
		cfg:       cfg,
		generator: generator,
		writer:    writer,
	}
}

// Run は、台本(MangaResponse)を受け取り、指定されたパネルの画像を生成するのだ。
func (r *MangaPanelImageRunner) Run(ctx context.Context, manga domain.MangaResponse, targetIndices []int) ([]*imagedom.ImageResponse, error) {
	// 1. 型を domain.Panel に合わせて修正するのだ
	allPanels := manga.Panels
	var targetPanels []domain.Panel

	// 2. 生成対象のフィルタリング
	if len(targetIndices) > 0 {
		slog.Info("Generating specific panels", "indices", targetIndices)
		for _, idx := range targetIndices {
			if idx >= 0 && idx < len(allPanels) {
				targetPanels = append(targetPanels, allPanels[idx])
			} else {
				slog.Warn("Index out of range, skipping", "index", idx, "total_panels", len(allPanels))
			}
		}
	} else {
		// 指定がない場合は全件（全パネル）対象なのだ
		targetPanels = allPanels
	}

	if len(targetPanels) == 0 {
		slog.Info("No panels to generate.")
		return []*imagedom.ImageResponse{}, nil
	}

	slog.Info("Starting parallel image generation",
		"title", manga.Title,
		"target_count", len(targetPanels),
		"total_count", len(allPanels),
	)

	images, err := r.generator.Execute(ctx, targetPanels)
	if err != nil {
		slog.Error("Image generation pipeline failed", "error", err)
		return nil, err
	}

	slog.Info("Successfully generated panels", "count", len(images))
	return images, nil
}

func (r *MangaPanelImageRunner) RunAndSave(ctx context.Context, manga domain.MangaResponse, targetIndices []int) ([]string, error) {
	// TODO::あとで画像ファイルとページ構成のjson出力
	return nil, nil
}
