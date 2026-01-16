package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/shouni/go-manga-kit/pkg/asset"
	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/domain"
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

// Run は、指定されたパスの Markdown コンテンツを解析し、漫画ページ画像（バイナリデータ）のリストを生成します。
func (r *MangaPageRunner) Run(ctx context.Context, plotPath string) ([]*imagedom.ImageResponse, error) {
	// 1. ファイル（JSON）の読み込み
	rc, err := r.reader.Open(ctx, plotPath)
	if err != nil {
		return nil, fmt.Errorf("プロットファイルのオープンに失敗しました (%s): %w", plotPath, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("プロットファイルの読み込みに失敗しました: %w", err)
	}

	// 2. JSONを構造体にデコード
	var manga domain.MangaResponse
	if err := json.Unmarshal(data, &manga); err != nil {
		return nil, fmt.Errorf("プロットデータのJSONパースに失敗しました: %w", err)
	}

	// 3. バリデーション
	if len(manga.Panels) == 0 {
		return nil, fmt.Errorf("マンガのパネルデータが空なのだ。生成を中止します")
	}

	slog.InfoContext(ctx, "MangaPageRunner: プロット読み込み完了",
		"path", plotPath,
		"title", manga.Title,
		"panelCount", len(manga.Panels),
	)

	// 4. ページ生成エンジン（を実行
	return r.generator.Execute(ctx, manga)
}

// RunAndSave は、画像の生成から指定ディレクトリへの保存までを一括で行います。
func (r *MangaPageRunner) RunAndSave(ctx context.Context, plotPath string, explicitOutputDir string) ([]string, error) {
	// 1. 保存先ディレクトリの決定
	targetDir := explicitOutputDir
	if targetDir == "" {
		// 明示的な出力ディレクトリが指定されていない場合、
		targetDir = asset.ResolveBaseURL(plotPath)
		if targetDir == "" {
			return nil, fmt.Errorf("アセットパスからベースURLを解決できませんでした: %s", plotPath)
		}
	}

	// 2. ベースとなる出力パスを解決します（GCS/ローカルを判別し、ベースファイル名を結合）
	basePath, err := asset.ResolveOutputPath(targetDir, asset.DefaultPageFileName)
	if err != nil {
		return nil, fmt.Errorf("出力パスの解決に失敗しました: %w", err)
	}

	// 3. 画像の生成
	resps, err := r.Run(ctx, plotPath)
	if err != nil {
		return nil, err // Run 内部でエラーラップされているためそのまま返す
	}

	// 4. 連番を付けて保存
	var savedPaths []string
	for i, resp := range resps {
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
