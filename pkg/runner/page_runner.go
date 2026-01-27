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

// MangaPageRunner は Markdown の解析、複数ページの画像生成、および成果物の保存を管理します。
type MangaPageRunner struct {
	cfg       config.Config
	generator generator.PagesImageGenerator
	reader    remoteio.InputReader
	writer    remoteio.OutputWriter
}

// NewMangaPageRunner は、設定、パーサー、生成エンジン、およびライターを依存性として注入し、MangaPageRunner を初期化します。
func NewMangaPageRunner(
	cfg config.Config,
	generator generator.PagesImageGenerator,
	reader remoteio.InputReader,
	writer remoteio.OutputWriter,
) *MangaPageRunner {
	return &MangaPageRunner{
		cfg:       cfg,
		generator: generator,
		reader:    reader,
		writer:    writer,
	}
}

// Run は、構造化された台本データを基に、最終的な漫画ページ画像を生成します。
func (r *MangaPageRunner) Run(ctx context.Context, manga *mangadom.MangaResponse) ([]*imagedom.ImageResponse, error) {
	// 1. バリデーション
	if manga == nil {
		return nil, fmt.Errorf("manga データが nil です")
	}

	// domain パッケージの定義に合わせて Pages か Panels かを確認してください
	if len(manga.Panels) == 0 {
		return nil, fmt.Errorf("プロットにページデータが含まれていません")
	}

	slog.InfoContext(ctx, "MangaPageRunner: ページ生成を開始します",
		"title", manga.Title,
		"pageCount", len(manga.Panels),
	)

	// 2. ページ生成エンジンを実行 (内部でレイアウトやテキスト合成を行う想定)
	// generator.Execute が *domain.MangaResponse を受け取れるようにします
	images, err := r.generator.Execute(ctx, manga)
	if err != nil {
		return nil, fmt.Errorf("ページ画像の生成に失敗しました: %w", err)
	}

	return images, nil
}

// RunAndSave は、画像の生成から指定ディレクトリへの保存までを一括で行います。
func (r *MangaPageRunner) RunAndSave(ctx context.Context, manga *mangadom.MangaResponse, plotPath string) ([]string, error) {
	if manga == nil {
		return nil, fmt.Errorf("manga データがありません")
	}

	// 1. 保存先ディレクトリの決定
	targetDir := asset.ResolveBaseURL(plotPath)
	if targetDir == "" {
		return nil, fmt.Errorf("アセットパスからベースURLを解決できませんでした: %s", plotPath)
	}

	// 2. ベースとなる出力パスを解決します（GCS/ローカルを判別し、ベースファイル名を結合）
	basePath, err := asset.ResolveOutputPath(targetDir, asset.DefaultPageImagePath())
	if err != nil {
		return nil, fmt.Errorf("出力パスの解決に失敗しました: %w", err)
	}

	// 3. 画像の生成
	// Run メソッドには、読み込み済みの manga オブジェクトを渡す形に合わせます
	resps, err := r.Run(ctx, manga)
	if err != nil {
		return nil, err
	}

	// 4. 連番を付けて保存
	var savedPaths []string
	for i, resp := range resps {
		// 例: manga_page.png -> manga_page_1.png
		pagePath, err := asset.GenerateIndexedPath(basePath, i+1)
		if err != nil {
			return nil, fmt.Errorf("ページ %d の出力パス生成に失敗しました: %w", i+1, err)
		}

		slog.InfoContext(ctx, "ページ画像を保存しています",
			"index", i+1,
			"path", pagePath,
		)

		if err := r.writer.Write(ctx, pagePath, bytes.NewReader(resp.Data), resp.MimeType); err != nil {
			return nil, fmt.Errorf("第 %d ページの保存に失敗しました (path: %s): %w", i+1, pagePath, err)
		}
		savedPaths = append(savedPaths, pagePath)
	}

	return savedPaths, nil
}
