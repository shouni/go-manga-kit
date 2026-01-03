package parser

import (
	"fmt"
	"strings"

	"github.com/shouni/gemini-image-kit/pkg/domain"
)

// Markdown解析で使用されるフィールドキーの定数定義
const (
	fieldKeySpeaker = "speaker"
	fieldKeyText    = "text"
	fieldKeyAction  = "action"
	fieldKeyLayout  = "layout"
)

// Parser はMarkdown形式の台本を解析し、構造化されたデータに変換する構造体です。
type Parser struct {
	baseURL string // GCS等のアセット参照用ベースURL
}

// NewParser は新しい Parser インスタンスを生成します。
// scriptURL が指定されている場合、そのディレクトリをベースURLとしてアセットのパスを解決します。
func NewParser(scriptURL string) *Parser {
	return &Parser{
		baseURL: resolveBaseURL(scriptURL),
	}
}

// Parse はMarkdownテキストを解析し、domain.MangaResponse 構造体に変換します。
func (p *Parser) Parse(input string) (*domain.MangaResponse, error) {
	manga := &domain.MangaResponse{}
	lines := strings.Split(input, "\n")
	var currentPage *domain.MangaPage

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		// 1. タイトル行 (# Title) の解析
		if m := TitleRegex.FindStringSubmatch(trimmedLine); m != nil {
			manga.Title = strings.TrimSpace(m[1])
			continue
		}

		// 2. パネル区切り (## Panel) の解析とアセットパスの解決
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

		// 3. フィールド行 (- key: value) の解析
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
				case fieldKeyLayout:
					// 将来的な拡張（レイアウト指定等）のために予約
				}
			}
		}
	}

	// 最後のパネルをスライスに追加
	if currentPage != nil {
		manga.Pages = append(manga.Pages, *currentPage)
	}

	if len(manga.Pages) == 0 {
		return nil, fmt.Errorf("有効なパネル情報が見つかりませんでした")
	}

	return manga, nil
}

// resolveFullPath は相対パスをbaseURLと結合し、完全なURLを生成します。
func (p *Parser) resolveFullPath(refPath string) string {
	if refPath == "" || strings.HasPrefix(refPath, "http") {
		return refPath
	}
	return p.baseURL + refPath
}

// extractReferencePath は行から ":" 以降の参照パスを抽出します。
func extractReferencePath(line string) string {
	_, after, found := strings.Cut(line, ":")
	if !found {
		return ""
	}
	return strings.TrimSpace(after)
}
