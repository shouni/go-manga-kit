package publisher

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/asset"
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
func (p *MangaPublisher) Publish(ctx context.Context, manga domain.MangaResponse, images []*imagedom.ImageResponse, opts Options) (PublishResult, error) {
	result := PublishResult{}

	// 1. 出力パスの解決
	markdown, err := asset.ResolveOutputPath(opts.OutputDir, asset.DefaultMangaPlotName)
	if err != nil {
		return result, err
	}
	result.MarkdownPath = markdown

	// 画像ディレクトリのベースパスを作成
	imgDir, err := asset.ResolveOutputPath(opts.OutputDir, asset.DefaultImageDir)
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
		relPath := path.Join(asset.DefaultImageDir, filepath.Base(pathStr))
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

		// Markdownファイルのパスから拡張子を置換し、HTMLファイルのパスを生成します。
		htmlPath := strings.TrimSuffix(markdown, filepath.Ext(markdown)) + ".html"
		if err := p.writer.Write(ctx, htmlPath, htmlBuffer, "text/html; charset=utf-8"); err != nil {
			return result, fmt.Errorf("HTMLファイルの書き込みに失敗しました: %w", err)
		}
		result.HTMLPath = htmlPath
	}

	return result, nil
}

// saveImages 指定されたディレクトリまたはリモートストレージ（GCS等）に画像データを保存し、保存先のパス一覧を返却します。
func (p *MangaPublisher) saveImages(ctx context.Context, images []*imagedom.ImageResponse, baseDir string) ([]string, error) {
	var paths []string
	for i, img := range images {
		if img == nil || len(img.Data) == 0 {
			continue
		}
		name, err := asset.GenerateIndexedPath(asset.DefaultPanelFileName, i+1)
		if err != nil {
			return nil, fmt.Errorf("パネル画像名の生成に失敗しました: %w", err)
		}
		fullPath, err := asset.ResolveOutputPath(baseDir, name)
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

// buildMarkdown 指定された漫画データ（manga）から、Markdown形式のコンテンツを生成して返します。
func (p *MangaPublisher) buildMarkdown(manga domain.MangaResponse, imagePaths []string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", manga.Title))
	for i, page := range manga.Pages {
		img := placeholder
		if i < len(imagePaths) {
			img = imagePaths[i]
		}
		if page.Dialogue != "" {
			sb.WriteString(fmt.Sprintf("## Panel: %s\n", img))
			character := p.characters.FindCharacter(page.SpeakerID)
			var speakerID string
			if character != nil {
				speakerID = character.ID
			} else {
				if primary := p.characters.GetPrimary(); primary != nil {
					speakerID = primary.ID
				}
			}

			sb.WriteString(fmt.Sprintf("- SpeakerID: %s\n", speakerID))
			sb.WriteString(fmt.Sprintf("- Dialogue: %s\n", strings.TrimSpace(tagRegex.ReplaceAllString(page.Dialogue, ""))))
			sb.WriteString(fmt.Sprintf("- VisualAnchor: %s\n", strings.TrimSpace(tagRegex.ReplaceAllString(page.VisualAnchor, ""))))
			//sb.WriteString("- layout: standard\n")
			//h.Reset()
			//h.Write([]byte(speakerID))
			//		speakerClass := "speaker-" + hex.EncodeToString(h.Sum(nil))[:10]
			//		sb.WriteString(fmt.Sprintf("- speakerclass: %s\n", speakerClass))
			//			sb.WriteString(p.getDialogueStyle(i))
			sb.WriteString("\n")
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
