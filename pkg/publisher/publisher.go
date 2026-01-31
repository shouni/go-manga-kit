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

	// 2. 構造体内の ReferenceURL から画像パスリストを作成
	imagePaths := make([]string, 0, len(manga.Panels))
	for _, panel := range manga.Panels {
		var relPath string
		if panel.ReferenceURL != "" {
			relPath = path.Join(asset.DefaultImageDir, filepath.Base(panel.ReferenceURL))
		}
		imagePaths = append(imagePaths, relPath)
	}
	result.ImagePaths = imagePaths
	content := p.buildMarkdown(manga, imagePaths)

	// 3. Markdown ファイルの書き出し
	slog.InfoContext(ctx, "Markdown ファイルを保存しています", "path", markdown)
	if err := p.writer.Write(ctx, markdown, strings.NewReader(content), "text/markdown; charset=utf-8"); err != nil {
		return result, fmt.Errorf("Markdown ファイルの書き込みに失敗: %w", err)
	}

	// 4. HTML 変換と保存
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

// buildMarkdown は画像、話者、セリフを含む Markdown を構築します。
func (p *MangaPublisher) buildMarkdown(manga *domain.MangaResponse, imagePaths []string) string {
	var sb strings.Builder

	// タイトルと説明文
	sb.WriteString(fmt.Sprintf("# %s\n\n", manga.Title))
	if manga.Description != "" {
		sb.WriteString(manga.Description + "\n\n")
	}

	// パネルごとの出力
	for i, panel := range manga.Panels {
		hasImage := i < len(imagePaths) && imagePaths[i] != ""
		hasDialogue := panel.Dialogue != ""

		// 画像もセリフもないパネル（ト書きのみ等）は、表示上のノイズになるためスキップ
		if !hasImage && !hasDialogue {
			continue
		}

		// 1. 画像が存在する場合のみ Markdown 記法を出力
		if hasImage {
			altText := panel.VisualAnchor
			if altText == "" {
				altText = fmt.Sprintf("Panel %d", i+1)
			}
			sb.WriteString(fmt.Sprintf("![%s](%s)\n\n", altText, imagePaths[i]))
		}

		// 2. セリフを出力
		if hasDialogue {
			if panel.SpeakerID != "" {
				// 話者名を強調
				sb.WriteString(fmt.Sprintf("**%s**: %s\n\n", panel.SpeakerID, panel.Dialogue))
			} else {
				sb.WriteString(fmt.Sprintf("%s\n\n", panel.Dialogue))
			}
		}

		// パネル間のセパレーター
		sb.WriteString("---\n\n")
	}

	return sb.String()
}
