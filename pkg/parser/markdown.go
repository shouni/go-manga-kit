package parser

import (
	"fmt"
	"log/slog"
	"path"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/publisher"
)

const (
	fieldKeySpeaker = "speaker"
	fieldKeyText    = "text"
	fieldKeyAction  = "action"
)

// Parser は解析するためのインターフェースなのだ。
type Parser interface {
	Parse(scriptURL string, input string) (*domain.MangaResponse, error)
}

type MarkdownParser struct{}

func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{}
}

func (p *MarkdownParser) Parse(scriptURL string, input string) (*domain.MangaResponse, error) {
	// 1. scriptURL (gs://.../manga.md) からディレクトリ部分を抽出する
	// path.Dir を使うことで、ファイル名を除いたベース部分が取得できるのだ
	baseURL := ""
	if scriptURL != "" && strings.Contains(scriptURL, "://") {
		baseURL = path.Dir(scriptURL)
	}

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

		// タイトル解析
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
			fullPath := publisher.ResolveFullPath(baseURL, refPath)

			currentPage = &domain.MangaPage{
				Page:         len(manga.Pages) + 1,
				ReferenceURL: fullPath,
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
					slog.Debug("Markdown内に未知のフィールドキーが見つかりました", "key", key)
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

func hasContent(page *domain.MangaPage) bool {
	return page.ReferenceURL != "" || page.Dialogue != "" || page.VisualAnchor != ""
}
