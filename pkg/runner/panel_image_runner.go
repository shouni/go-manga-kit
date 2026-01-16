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

// Run は、台本(MangaResponse)を受け取り、パネルの画像を生成するのだ。
// func (r *MangaPanelImageRunner) Run(ctx context.Context, manga mangadom.MangaResponse, targetIndices []int) ([]*imagedom.ImageResponse, error) {
func (r *MangaPanelImageRunner) Run(ctx context.Context, manga *mangadom.MangaResponse) ([]*imagedom.ImageResponse, error) {
	slog.Info("Starting parallel image generation")

	images, err := r.generator.Execute(ctx, manga.Panels)
	if err != nil {
		slog.Error("Image generation pipeline failed", "error", err)
		return nil, err
	}

	slog.Info("Successfully generated panels", "count", len(images))
	return images, nil
}

// RunAndSave 画像パネルを生成し、インデックスを付けて指定のパスに保存します。エラーを返します。
func (r *MangaPanelImageRunner) RunAndSave(ctx context.Context, manga *mangadom.MangaResponse, scriptPath string) error {
	// 保存先ディレクトリの決定
	targetDir := asset.ResolveBaseURL(scriptPath)

	// ベースとなる出力パスを解決します（GCS/ローカルを判別し、ベースファイル名を結合）
	basePath, err := asset.ResolveOutputPath(targetDir, asset.DefaultPanelFileName)
	if err != nil {
		return fmt.Errorf("出力パスの解決に失敗しました: %w", err)
	}

	// 画像の生成
	images, err := r.Run(ctx, manga)
	if err != nil {
		return err // Run 内部でエラーラップされているためそのまま返す
	}

	if len(images) != len(manga.Panels) {
		return fmt.Errorf("生成された画像の数(%d)とパネルの数(%d)が一致しません", len(images), len(manga.Panels))
	}
	for i, image := range images {
		// 連番を付けて保存
		panelPath, err := asset.GenerateIndexedPath(basePath, i+1)
		if err != nil {
			return fmt.Errorf("パネル %d の出力パス生成に失敗しました: %w", i+1, err)
		}

		slog.InfoContext(ctx, "パネル画像を保存しています",
			"index", i+1,
			"path", panelPath,
		)

		if err := r.writer.Write(ctx, panelPath, bytes.NewReader(image.Data), image.MimeType); err != nil {
			// エラー発生時は、それまでの成果物は返さず、nilとエラーを返す
			return fmt.Errorf("第 %d パネルの保存に失敗しました (path: %s): %w", i+1, panelPath, err)
		}
		manga.Panels[i].ReferenceURL = panelPath
	}

	return nil
}
