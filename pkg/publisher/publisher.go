package publisher

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/asset"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"github.com/shouni/go-text-format/pkg/md2htmlrunner"
)

// markdownEscaper は Markdown の制御文字と HTML タグ文字を効率的にエスケープするための Replacer です。
var markdownEscaper = strings.NewReplacer(
	"*", "\\*",
	"_", "\\_",
	"[", "\\[",
	"]", "\\]",
	"#", "\\#",
	"`", "\\`",
	"<", "&lt;",
	">", "&gt;",
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

// NewMangaPublisher は新しいインスタンスを作成します。
func NewMangaPublisher(writer remoteio.OutputWriter, htmlRunner md2htmlrunner.Runner) *MangaPublisher {
	return &MangaPublisher{
		writer:     writer,
		htmlRunner: htmlRunner,
	}
}

// Publish はドメインモデルを基に Markdown を構築し、HTML への変換・保存を実行します。
func (p *MangaPublisher) Publish(ctx context.Context, manga *domain.MangaResponse, opts Options) (PublishResult, error) {
	result := PublishResult{}
	if manga == nil {
		return result, fmt.Errorf("manga データが nil です")
	}

	markdownPath, err := asset.ResolveOutputPath(opts.OutputDir, asset.DefaultMangaPlotName)
	if err != nil {
		return result, fmt.Errorf("Markdown 出力パスの解決に失敗: %w", err)
	}
	result.MarkdownPath = markdownPath

	// 保存用に相対パスリストを作成
	imagePaths := make([]string, 0, len(manga.Panels))
	for _, panel := range manga.Panels {
		var relPath string
		if panel.ReferenceURL != "" {
			relPath = path.Join(asset.DefaultImageDir, path.Base(panel.ReferenceURL))
		}
		imagePaths = append(imagePaths, relPath)
	}
	result.ImagePaths = imagePaths

	// Markdown 文字列の構築
	content := p.BuildMarkdownOnly(manga, imagePaths)

	// Markdown の保存
	slog.InfoContext(ctx, "Markdown ファイルを保存しています", "path", markdownPath)
	if err := p.writer.Write(ctx, markdownPath, strings.NewReader(content), "text/markdown; charset=utf-8"); err != nil {
		return result, fmt.Errorf("Markdown 書き込み失敗: %w", err)
	}

	// HTML の生成
	if p.htmlRunner != nil {
		htmlBuffer, err := p.htmlRunner.Run(ctx, manga.Title, []byte(content))
		if err != nil {
			return result, fmt.Errorf("HTML 変換失敗: %w", err)
		}
		htmlPath := strings.TrimSuffix(markdownPath, path.Ext(markdownPath)) + ".html"
		if err := p.writer.Write(ctx, htmlPath, htmlBuffer, "text/html; charset=utf-8"); err != nil {
			return result, fmt.Errorf("HTML 書き込み失敗: %w", err)
		}
		result.HTMLPath = htmlPath
	}

	return result, nil
}

// BuildMarkdownOnly は画像、話者、セリフを含む Markdown 文字列のみを生成して返却します。
// imagePaths が指定されている場合はそちらを優先し、nil の場合は構造体内の ReferenceURL を使用します。
func (p *MangaPublisher) BuildMarkdownOnly(manga *domain.MangaResponse, imagePaths []string) string {
	var sb strings.Builder

	// タイトルと説明文
	sb.WriteString(fmt.Sprintf("# %s\n\n", escapeMarkdown(manga.Title)))
	if manga.Description != "" {
		sb.WriteString(escapeMarkdown(manga.Description) + "\n\n")
	}

	firstPanel := true
	for i, panel := range manga.Panels {
		var currentImagePath string
		// 引数 imagePaths があれば優先、なければ構造体内の URL を使う
		if i < len(imagePaths) {
			currentImagePath = imagePaths[i]
		} else {
			currentImagePath = panel.ReferenceURL
		}

		hasImage := currentImagePath != ""
		hasDialogue := panel.Dialogue != ""
		if !hasImage && !hasDialogue {
			continue
		}

		if !firstPanel {
			sb.WriteString("---\n\n")
		}
		firstPanel = false

		if hasImage {
			altText := panel.VisualAnchor
			if altText == "" {
				altText = fmt.Sprintf("Panel %d", i+1)
			}
			sb.WriteString(fmt.Sprintf("![%s](%s)\n\n", escapeMarkdown(altText), currentImagePath))
		}

		if hasDialogue {
			dialogue := escapeMarkdown(panel.Dialogue)
			if panel.SpeakerID != "" {
				sb.WriteString(fmt.Sprintf("**%s**: %s\n\n", escapeMarkdown(panel.SpeakerID), dialogue))
			} else {
				sb.WriteString(fmt.Sprintf("%s\n\n", dialogue))
			}
		}
	}

	return sb.String()
}

// escapeMarkdown は Markdown の制御文字と HTML 特殊文字を安全に置換します。
func escapeMarkdown(text string) string {
	return markdownEscaper.Replace(text)
}
