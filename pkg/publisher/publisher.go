package publisher

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"regexp"
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

const (
	placeholder     = "placeholder.png"
	evenPanelTail   = "top"
	evenPanelBottom = "10%"
	evenPanelLeft   = "10%"
	oddPanelTail    = "bottom"
	oddPanelTop     = "10%"
	oddPanelRight   = "10%"
)

var tagRegex = regexp.MustCompile(`\[[^\]]+\]`)

// MangaPublisher は成果物の永続化とフォーマット変換を担います。
type MangaPublisher struct {
	characters domain.CharactersMap
	writer     remoteio.OutputWriter
	htmlRunner md2htmlrunner.Runner
}

// NewMangaPublisher は、指定された依存関係を持つMangaPublisherの新しいインスタンスを作成して返却します。
func NewMangaPublisher(
	characters domain.CharactersMap,
	writer remoteio.OutputWriter,
	htmlRunner md2htmlrunner.Runner,
) *MangaPublisher {
	return &MangaPublisher{
		characters: characters,
		writer:     writer,
		htmlRunner: htmlRunner,
	}
}

// Publish は画像の保存、Markdownの構築、HTML変換を一括して実行し、生成されたファイル情報を返却します。
// Publish は既存の画像パス（ReferenceURL）を参照して Markdown を構築し、HTML への変換・保存を実行します。
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
		// ReferenceURL (例: gs://bucket/images/panel_1.png) からファイル名を取得し
		// images/panel_1.png のような相対パスを構築します
		relPath := path.Join(asset.DefaultImageDir, filepath.Base(panel.ReferenceURL))
		imagePaths = append(imagePaths, relPath)
	}
	result.ImagePaths = imagePaths

	// 3. Markdown テキストの構築
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

// buildMarkdown は漫画データと画像相対パスから Markdown 形式の文字列を生成します。
func (p *MangaPublisher) buildMarkdown(manga *domain.MangaResponse, imagePaths []string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", manga.Title))

	if manga.Description != "" {
		sb.WriteString(fmt.Sprintf("> %s\n\n", manga.Description))
	}

	for i, panel := range manga.Panels {
		img := placeholder
		if i < len(imagePaths) && imagePaths[i] != "" {
			img = imagePaths[i]
		}

		// パネル情報の出力（セリフかアンカーがある場合）
		if panel.Dialogue != "" || panel.VisualAnchor != "" {
			// 1. パネル見出しと画像
			sb.WriteString(fmt.Sprintf("## Panel %d\n", i+1))
			sb.WriteString(fmt.Sprintf("![Panel Image](%s)\n\n", img))

			//// 2. スタイル・設定（HTML変換用メタデータ）
			//// リスト形式ではなく、あえて独立したセクション、あるいは特定の記法にすることで
			//// md2html側で正規表現などで一括置換・スタイル適用しやすくします
			//sb.WriteString("### Metadata\n")
			//sb.WriteString(p.getDialogueStyle(i))

			// 3. 話者とセリフ（引用符を使うことでシナリオ感を演出）
			character := p.characters.GetCharacterWithDefault(panel.SpeakerID)
			speaker := panel.SpeakerID
			if character != nil {
				speaker = character.Name
			}

			sb.WriteString(fmt.Sprintf("- **Speaker**: %s\n", speaker))

			if panel.Dialogue != "" {
				cleanDialogue := strings.TrimSpace(tagRegex.ReplaceAllString(panel.Dialogue, ""))
				// セリフを引用ブロックにすることで、HTML化の際に見栄えが良くなります
				sb.WriteString(fmt.Sprintf("- **Dialogue**: \n> %s\n", cleanDialogue))
			}

			if panel.VisualAnchor != "" {
				cleanAnchor := strings.TrimSpace(tagRegex.ReplaceAllString(panel.VisualAnchor, ""))
				// 描画指示は補足情報としてイタリックに
				sb.WriteString(fmt.Sprintf("- *Visual Anchor*: %s\n", cleanAnchor))
			}

			sb.WriteString("\n---\n\n") // パネルごとの区切り線
		}
	}
	return sb.String()
}

// getDialogueStyle returns the style for the specified panel's dialogue.
func (p *MangaPublisher) getDialogueStyle(idx int) string {
	if idx%2 == 0 {
		return fmt.Sprintf("- tail: %s\n- bottom: %s\n- left: %s\n", evenPanelTail, evenPanelBottom, evenPanelLeft)
	}
	return fmt.Sprintf("- tail: %s\n- top: %s\n- right: %s\n", oddPanelTail, oddPanelTop, oddPanelRight)
}
