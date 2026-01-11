package publisher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
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
	defaultMangaPlotName = "manga_plot.md"
	defaultImageDirName  = "images"
	placeholder          = "placeholder.png"
	evenPanelTail        = "top"
	evenPanelBottom      = "10%"
	evenPanelLeft        = "10%"
	oddPanelTail         = "bottom"
	oddPanelTop          = "10%"
	oddPanelRight        = "10%"
	defaultNarrationName = "narration"
)

var tagRegex = regexp.MustCompile(`\[[^\]]+\]`)

// MangaPublisher は成果物の永続化とフォーマット変換を担います。
type MangaPublisher struct {
	writer     remoteio.OutputWriter
	htmlRunner md2htmlrunner.Runner
}

// NewMangaPublisher creates and returns a new instance of MangaPublisher with the specified writer and HTML runner.
func NewMangaPublisher(writer remoteio.OutputWriter, htmlRunner md2htmlrunner.Runner) *MangaPublisher {
	return &MangaPublisher{
		writer:     writer,
		htmlRunner: htmlRunner,
	}
}

// Publish は画像の保存、Markdownの構築、HTML変換を一括して実行し、生成されたファイル情報を返却するのだ！
func (p *MangaPublisher) Publish(ctx context.Context, manga domain.MangaResponse, images []*imagedom.ImageResponse, opts Options) (PublishResult, error) {
	result := PublishResult{}

	// 1. 出力パスの解決
	markdown, err := ResolveOutputPath(opts.OutputDir, defaultMangaPlotName)
	if err != nil {
		return result, err
	}
	result.MarkdownPath = markdown

	// 画像ディレクトリのベースパスを作成
	imgDir, err := ResolveOutputPath(opts.OutputDir, defaultImageDirName)
	if err != nil {
		return result, err
	}

	// 2. 画像の保存
	savedPaths, err := p.saveImages(ctx, images, imgDir)
	if err != nil {
		return result, fmt.Errorf("画像の書き込みに失敗しました: %w", err)
	}
	result.ImagePaths = savedPaths

	// 3. Markdown用相対パスの作成
	relativePaths := make([]string, 0, len(savedPaths))
	for _, pathStr := range savedPaths {
		relPath := path.Join(defaultImageDirName, filepath.Base(pathStr))
		relativePaths = append(relativePaths, relPath)
	}

	// 4. Markdownテキストの構築
	content := p.buildMarkdown(manga, relativePaths)

	// 5. Markdownファイルの書き出し
	if err := p.writer.Write(ctx, markdown, strings.NewReader(content), "text/markdown; charset=utf-8"); err != nil {
		return result, fmt.Errorf("markdownファイルの書き込みに失敗しました: %w", err)
	}

	// 6. HTML変換と保存
	if p.htmlRunner != nil {
		slog.Info("Converting to Webtoon HTML", "title", manga.Title)
		htmlBuffer, err := p.htmlRunner.Run(ctx, manga.Title, []byte(content))
		if err != nil {
			return result, fmt.Errorf("HTMLの変換に失敗しました: %w", err)
		}

		// Markdownの拡張子を置換してHTMLパスを生成するのだ
		htmlPath := strings.TrimSuffix(markdown, filepath.Ext(markdown)) + ".html"
		if err := p.writer.Write(ctx, htmlPath, htmlBuffer, "text/html; charset=utf-8"); err != nil {
			return result, fmt.Errorf("markdownファイルの書き込みに失敗しました: : %w", err)
		}
		result.HTMLPath = htmlPath
	}

	return result, nil
}

// saveImages saves image data to the specified directory or remote storage (e.g., GCS) and returns their paths.
func (p *MangaPublisher) saveImages(ctx context.Context, images []*imagedom.ImageResponse, baseDir string) ([]string, error) {
	var paths []string
	for i, img := range images {
		if img == nil || len(img.Data) == 0 {
			continue
		}
		name := fmt.Sprintf("panel_%d.png", i+1)
		fullPath, err := ResolveOutputPath(baseDir, name)
		if err != nil {
			return nil, fmt.Errorf("出力パスの解決に失敗しました: %w", err)
		}

		if err := p.writer.Write(ctx, fullPath, bytes.NewReader(img.Data), "image/png"); err != nil {
			return nil, fmt.Errorf("画像の書き込みに失敗しました %s: %w", fullPath, err)
		}
		paths = append(paths, fullPath)
	}
	return paths, nil
}

// buildMarkdown returns the Markdown content for the specified manga.
func (p *MangaPublisher) buildMarkdown(manga domain.MangaResponse, imagePaths []string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", manga.Title))
	h := sha256.New()

	for i, page := range manga.Pages {
		img := placeholder
		if i < len(imagePaths) {
			img = imagePaths[i]
		}

		sb.WriteString(fmt.Sprintf("## Panel: %s\n", img))
		sb.WriteString("- layout: standard\n")

		if page.Dialogue != "" {
			speaker := page.SpeakerID
			if speaker == "" {
				speaker = defaultNarrationName
			}
			text := strings.TrimSpace(tagRegex.ReplaceAllString(page.Dialogue, ""))

			h.Reset()
			h.Write([]byte(speaker))
			speakerClass := "speaker-" + hex.EncodeToString(h.Sum(nil))[:10]

			sb.WriteString(fmt.Sprintf("- speaker: %s\n", speakerClass))
			sb.WriteString(fmt.Sprintf("- text: %s\n", text))
			sb.WriteString(p.getDialogueStyle(i))
		} else {
			sb.WriteString("- type: none\n")
		}
		sb.WriteString("\n")
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
