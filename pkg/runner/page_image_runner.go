package runner

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/shouni/go-manga-kit/pkg/asset"
	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-manga-kit/pkg/parser"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

// MangaPageRunner は Markdown の解析、複数ページの画像生成、および成果物の保存を管理します。
type MangaPageRunner struct {
	cfg      config.Config
	mkParser parser.Parser
	pageGen  *generator.PageGenerator
	writer   remoteio.OutputWriter
}

// NewMangaPageRunner は、設定、パーサー、生成エンジン、およびライターを依存性として注入し、MangaPageRunner を初期化します。
func NewMangaPageRunner(
	cfg config.Config,
	mkParser parser.Parser,
	mangaGen generator.MangaGenerator,
	writer remoteio.OutputWriter,
) *MangaPageRunner {
	return &MangaPageRunner{
		cfg:      cfg,
		mkParser: mkParser,
		pageGen:  generator.NewPageGenerator(mangaGen),
		writer:   writer,
	}
}

// Run は、指定されたパスの Markdown コンテンツを解析し、漫画ページ画像（バイナリデータ）のリストを生成します。
func (r *MangaPageRunner) Run(ctx context.Context, markdownPath string) ([]*imagedom.ImageResponse, error) {
	manga, err := r.mkParser.ParseFromPath(ctx, markdownPath)
	if err != nil {
		return nil, fmt.Errorf("markdown コンテンツの解析に失敗しました: %w", err)
	}
	if manga == nil {
		return nil, fmt.Errorf("マンガの解析結果が nil です")
	}

	slog.InfoContext(ctx, "MangaPageRunner: 解析完了",
		"path", markdownPath,
		"title", manga.Title,
		"panelCount", len(manga.Pages),
	)

	// ページ生成エンジンを実行し、画像レスポンス群を取得
	return r.pageGen.ExecuteMangaPages(ctx, *manga)
}

// RunAndSave は、画像の生成から指定ディレクトリへの保存までを一括で行います。
func (r *MangaPageRunner) RunAndSave(ctx context.Context, markdownPath string, explicitOutputDir string) ([]string, error) {
	// 1. 保存先ディレクトリの決定
	targetDir := explicitOutputDir
	if targetDir == "" {
		// 明示的な出力ディレクトリが指定されていない場合、
		// 入力されたMarkdownファイルと同じディレクトリを保存先とします。
		targetDir = asset.ResolveBaseURL(markdownPath)
		if targetDir == "" {
			return nil, fmt.Errorf("アセットパスからベースURLを解決できませんでした: %s", markdownPath)
		}
	}

	// 2. ベースとなる出力パスを解決します（GCS/ローカルを判別し、ベースファイル名を結合）
	basePath, err := asset.ResolveOutputPath(targetDir, asset.DefaultPageFileName)
	if err != nil {
		return nil, fmt.Errorf("出力パスの解決に失敗しました: %w", err)
	}

	// 3. 画像の生成
	resps, err := r.Run(ctx, markdownPath)
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
