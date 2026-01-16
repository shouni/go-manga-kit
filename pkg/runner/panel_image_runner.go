package runner

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/shouni/go-manga-kit/pkg/asset"
	"github.com/shouni/go-manga-kit/pkg/config"
	mangadom "github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/generator"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
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
func (r *MangaPanelImageRunner) Run(ctx context.Context, manga mangadom.MangaResponse, targetIndices []int) ([]*imagedom.ImageResponse, error) {
	// 1. 型を domain.Panel に合わせて修正するのだ
	allPanels := manga.Panels
	var targetPanels []mangadom.Panel

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

// RunAndSave 画像パネルを生成し、インデックスを付けて指定のパスに保存します。保存されたパス、またはエラーを返します。
func (r *MangaPanelImageRunner) RunAndSave(ctx context.Context, manga mangadom.MangaResponse, targetIndices []int, plotFile string) ([]string, error) {
	// TODO:パネルのターゲット指定
	var targetPanels []mangadom.Panel
	images, err := r.generator.Execute(ctx, targetPanels)
	if err != nil {
		slog.Error("Image generation pipeline failed", "error", err)
		return nil, err
	}

	slog.Info("Successfully generated panels", "count", len(images))
	// 4. 連番を付けて保存
	basePath, err := asset.ResolveOutputPath(plotFile, asset.DefaultPanelFileName)
	var savedPaths []string
	for i, resp := range images {
		// manga_page.png -> manga_page_1.png のように変換する
		pagePath, err := asset.GenerateIndexedPath(basePath, i+1)
		if err != nil {
			return nil, fmt.Errorf("ページ %d の出力パス生成に失敗しました: %w", i+1, err)
		}

		slog.InfoContext(ctx, "ページ画像を保存しています",
			"index", i+1,
			"path", pagePath,
		)

		if err := r.writer.Write(ctx, pagePath, bytes.NewReader(resp.Data), resp.MimeType); err != nil {
			// エラー発生時は、それまでの成果物は返さず、nilとエラーを返す
			return nil, fmt.Errorf("第 %d ページの保存に失敗しました (path: %s): %w", i+1, pagePath, err)
		}
		savedPaths = append(savedPaths, pagePath)
	}

	return savedPaths, nil
}
