package parser

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/asset"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

const (
	fieldKeySpeaker = "SpeakerID"
	fieldKeyText    = "Dialogue"
	fieldKeyAction  = "VisualAnchor"
)

// Parser は解析するためのインターフェースを定義します。
type Parser interface {
	ParseFromPath(ctx context.Context, fullPath string) (*domain.MangaResponse, error)
	Parse(input string, baseDir string) (*domain.MangaResponse, error)
}

// MarkdownParser は Markdown 形式の台本を解析する構造体です。
type MarkdownParser struct {
	reader remoteio.InputReader
}

// NewMarkdownParser は新しい MarkdownParser インスタンスを生成します。
func NewMarkdownParser(r remoteio.InputReader) *MarkdownParser {
	return &MarkdownParser{reader: r}
}

// ParseFromPath は指定された markdownAssetPath（GCS URIやローカルファイルパスなど）から
// コンテンツを読み込み、解析して domain.MangaResponse を返します。
func (p *MarkdownParser) ParseFromPath(ctx context.Context, markdownAssetPath string) (*domain.MangaResponse, error) {
	rc, err := p.reader.Open(ctx, markdownAssetPath)
	if err != nil {
		return nil, fmt.Errorf("台本ソースの読み込みに失敗しました (%s): %w", markdownAssetPath, err)
	}
	defer rc.Close()

	// リーダーのコンテンツをバッファに読み込みます。
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, rc); err != nil {
		return nil, fmt.Errorf("読み込み中のコンテンツコピーに失敗しました: %w", err)
	}

	// fullPath からディレクトリ部分（baseDir）を割り出すのだ
	baseDir := asset.ResolveBaseURL(markdownAssetPath)

	return p.Parse(buf.String(), baseDir)
}

// Parse は指定された Markdown テキストを解析します。
func (p *MarkdownParser) Parse(input string, baseDir string) (*domain.MangaResponse, error) {
	manga := &domain.MangaResponse{}
	lines := strings.Split(input, "\n")
	var currentPage *domain.MangaPage

	addPreviousPage := func() {
		if currentPage != nil && hasContent(currentPage) {
			manga.Pages = append(manga.Pages, *currentPage)
		}
	}

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		// タイトル解析 (# Title)
		if m := TitleRegex.FindStringSubmatch(trimmedLine); m != nil {
			manga.Title = strings.TrimSpace(m[1])
			continue
		}

		// パネル解析 (## Panel: path/to/image.png)
		if m := PanelRegex.FindStringSubmatch(trimmedLine); m != nil {
			addPreviousPage()

			var refPath string
			if len(m) > 1 {
				refPath = strings.TrimSpace(m[1])
			}
			resolvedFullPath, err := asset.ResolveOutputPath(baseDir, refPath)
			if err != nil {
				// パス解決の失敗は処理継続不可能なため、エラーをラップして返します。
				return nil, fmt.Errorf("panel画像のパス解決に失敗しました (base: %s, ref: %s): %w", baseDir, refPath, err)
			}
			currentPage = &domain.MangaPage{
				Page:         len(manga.Pages) + 1,
				ReferenceURL: resolvedFullPath,
			}
			continue
		}

		// フィールド行の解析
		if currentPage != nil {
			if m := FieldRegex.FindStringSubmatch(trimmedLine); m != nil {
				key, val := strings.ToLower(m[1]), strings.TrimSpace(m[2])
				switch key {
				case fieldKeySpeaker:
					currentPage.SpeakerID = strings.ToLower(val)
				case fieldKeyText:
					currentPage.Dialogue = val
				case fieldKeyAction:
					currentPage.VisualAnchor = val
				default:
					slog.Debug("未知のフィールドキーをスキップしました", "key", key)
				}
			}
		}
	}

	addPreviousPage()

	if len(manga.Pages) == 0 {
		return nil, fmt.Errorf("有効なパネル情報が見つかりませんでした")
	}

	return manga, nil
}

// hasContent はページが有効な情報を保持しているか判定します。
func hasContent(page *domain.MangaPage) bool {
	return page.ReferenceURL != "" || page.Dialogue != "" || page.VisualAnchor != ""
}
