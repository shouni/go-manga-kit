package parser

import (
	"fmt"
	"strings"

	"github.com/shouni/gemini-image-kit/pkg/domain"
)

// Parser はMarkdown形式の台本を解析する構造体なのだ
type Parser struct {
	baseURL string // GCS等のアセット参照用ベースURL
}

// NewParser は新しい Parser インスタンスを生成するのだ
// scriptURL が指定されている場合、そのディレクトリをベースURLとしてアセットを解決するのだ
func NewParser(scriptURL string) *Parser {
	return &Parser{
		baseURL: resolveBaseURL(scriptURL),
	}
}

// Parse はMarkdownテキストを MangaResponse 構造体に変換するのだ
func (p *Parser) Parse(input string) (*domain.MangaResponse, error) {
	manga := &domain.MangaResponse{}
	lines := strings.Split(input, "\n")
	var currentPage *domain.MangaPage

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		// 1. タイトル行 (# Title)
		if m := TitleRegex.FindStringSubmatch(trimmedLine); m != nil {
			manga.Title = strings.TrimSpace(m[1])
			continue
		}

		// 2. パネル区切り (## Panel)
		if PanelRegex.MatchString(trimmedLine) {
			if currentPage != nil {
				manga.Pages = append(manga.Pages, *currentPage)
			}

			refPath := extractReferencePath(trimmedLine)
			fullPath := p.resolveFullPath(refPath)

			currentPage = &domain.MangaPage{
				Page:         len(manga.Pages) + 1,
				ReferenceURL: fullPath,
			}
			continue
		}

		// 3. フィールド行 (- key: value)
		if currentPage != nil {
			if m := FieldRegex.FindStringSubmatch(trimmedLine); m != nil {
				key, val := strings.ToLower(m[1]), strings.TrimSpace(m[2])
				switch key {
				case "speaker":
					currentPage.SpeakerID = strings.ToLower(val)
				case "text":
					currentPage.Dialogue = val
				case "action":
					currentPage.VisualAnchor = val
				case "layout":
					// 将来的にレイアウト指定もパースできるように拡張可能なのだ
				}
			}
		}
	}

	if currentPage != nil {
		manga.Pages = append(manga.Pages, *currentPage)
	}

	if len(manga.Pages) == 0 {
		return nil, fmt.Errorf("有効なパネルが見つかりませんでしたなのだ")
	}

	return manga, nil
}

// resolveFullPath は相対パスをbaseURLと結合してフルURLにするのだ
func (p *Parser) resolveFullPath(refPath string) string {
	if refPath == "" || strings.HasPrefix(refPath, "http") {
		return refPath
	}
	return p.baseURL + refPath
}

// extractReferencePath は "## Panel: path/to/ref.png" からパス部分だけを抜き出すのだ
func extractReferencePath(line string) string {
	if !strings.Contains(line, ":") {
		return ""
	}
	_, after, found := strings.Cut(line, ":")
	if !found {
		return ""
	}
	return strings.TrimSpace(after)
}
