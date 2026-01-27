package publisher

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/asset"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"github.com/shouni/go-text-format/pkg/md2htmlrunner"
)

// Options はパブリッシュ動作を制御する設定項目です。
type Options struct {
	OutputDir string
}

// PublishResult はパブリッシュ処理の結果として生成されたファイルの情報を保持します。
type PublishResult struct {
	MarkdownPath string   // 生成された manga_plot.md のパス
	HTMLPath     string   // 生成された HTML のパス
	ImagePaths   []string // 保存された全画像のパスリスト
}

const placeholder = "placeholder.png"

// MangaPublisher は成果物の永続化とフォーマット変換を担います。
type MangaPublisher struct {
	writer     remoteio.OutputWriter
	htmlRunner md2htmlrunner.Runner
}

// NewMangaPublisher は、指定された依存関係を持つMangaPublisherの新しいインスタンスを作成して返却します。
func NewMangaPublisher(
	writer remoteio.OutputWriter,
	htmlRunner md2htmlrunner.Runner,
) *MangaPublisher {
	return &MangaPublisher{
		writer:     writer,
		htmlRunner: htmlRunner,
	}
}

// Publish はドメインモデルを基に Webtoon 形式の Markdown を構築し、HTML への変換・保存を実行します。
func (p *MangaPublisher) Publish(ctx context.Context, manga *domain.MangaResponse, opts Options) (PublishResult, error) {
	result := PublishResult{}

	if manga == nil {
		return result, fmt.Errorf("manga データが nil です")
	}

	// 1. 出力パス（Markdown）の解決
	markdown, err := asset.ResolveOutputPath(opts.OutputDir, asset.DefaultMangaPlotName)
	if err != nil {
		return result, fmt.Errorf("Markdown 出力パスの解決に失敗: %w", err)
	}
	result.MarkdownPath = markdown

	// 2. 構造体内の ReferenceURL から Markdown 用の相対パスリストを作成
	imagePaths := make([]string, 0, len(manga.Panels))
	for _, panel := range manga.Panels {
		// ReferenceURL からファイル名を取得し、相対パスを構築
		relPath := path.Join(asset.DefaultImageDir, filepath.Base(panel.ReferenceURL))
		imagePaths = append(imagePaths, relPath)
	}
	result.ImagePaths = imagePaths
	content := p.buildMarkdown(manga, imagePaths)

	// 4. Markdown ファイルの書き出し
	slog.InfoContext(ctx, "Markdown ファイルを保存しています", "path", markdown)
	if err := p.writer.Write(ctx, markdown, strings.NewReader(content), "text/markdown; charset=utf-8"); err != nil {
		return result, fmt.Errorf("Markdown ファイルの書き込みに失敗: %w", err)
	}

	// 5. HTML 変換と保存
	if p.htmlRunner != nil {
		slog.InfoContext(ctx, "HTML への変換を開始します", "title", manga.Title)
		htmlBuffer, err := p.htmlRunner.Run(ctx, manga.Title, []byte(content))
		if err != nil {
			return result, fmt.Errorf("HTML 変換に失敗: %w", err)
		}

		// Markdown のパスをベースに .html 拡張子へ置換
		htmlPath := strings.TrimSuffix(markdown, filepath.Ext(markdown)) + ".html"
		if err := p.writer.Write(ctx, htmlPath, htmlBuffer, "text/html; charset=utf-8"); err != nil {
			return result, fmt.Errorf("HTML ファイルの書き込みに失敗: %w", err)
		}
		result.HTMLPath = htmlPath
	}

	return result, nil
}

// buildMarkdown は WebtoonParser が解析可能な「純粋な画像リスト」形式の Markdown を生成します。
func (p *MangaPublisher) buildMarkdown(manga *domain.MangaResponse, imagePaths []string) string {
	var sb strings.Builder

	// タイトルを出力
	sb.WriteString(fmt.Sprintf("# %s\n\n", manga.Title))

	// 説明文を引用符なしのプレーンテキストで出力（WebtoonParser が Description として抽出）
	if manga.Description != "" {
		sb.WriteString(manga.Description + "\n\n")
	}

	// パネルを画像記法として出力
	for i, panel := range manga.Panels {
		img := placeholder
		if i < len(imagePaths) && imagePaths[i] != "" {
			img = imagePaths[i]
		}

		// Altテキストには VisualAnchor (描画指示) を活用し、アクセシビリティを高める
		altText := panel.VisualAnchor
		if altText == "" {
			altText = fmt.Sprintf("Panel %d", i+1)
		}

		// セリフや話者情報のテキスト出力は、Webtoonの没入感を損なうためここでは意図的に除外。
		// 画像（文字入り画像である前提）のみを美しく並べる形式にします。
		sb.WriteString(fmt.Sprintf("![%s](%s)\n", altText, img))
	}

	return sb.String()
}
