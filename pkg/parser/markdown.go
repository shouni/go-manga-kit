package parser

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"
)

const (
	fieldKeySpeaker = "speaker"
	fieldKeyText    = "text"
	fieldKeyAction  = "action"
	fieldKeyLayout  = "layout"
)

// Parser は解析するためのインターフェースなのだ。
type Parser interface {
	// Parse はスクリプトのURLと内容を受け取り、構造化された MangaResponse を返すのだ。
	Parse(scriptURL string, input string) (*domain.MangaResponse, error)
}

// MarkdownParser はMarkdown形式を解析し、構造化データに変換する構造体です。
type MarkdownParser struct {
}

// NewMarkdownParser は Parser を初期化するのだ。引数は不要になったのだ。
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{}
}

// Parse は指定された scriptURL を基に参照パスを解決し、Markdown テキストを解析して domain.MangaResponse 構造体に変換します。
func (p *MarkdownParser) Parse(scriptURL string, input string) (*domain.MangaResponse, error) {
	// 1. その時の scriptURL に基づいてベースURLを算出する
	baseURL := resolveBaseURL(scriptURL)

	manga := &domain.MangaResponse{}
	lines := strings.Split(input, "\n")
	var currentPage *domain.MangaPage

	// 前のページを確定して追加するヘルパー関数
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

		if m := TitleRegex.FindStringSubmatch(trimmedLine); m != nil {
			manga.Title = strings.TrimSpace(m[1])
			continue
		}

		if m := PanelRegex.FindStringSubmatch(trimmedLine); m != nil {
			addPreviousPage()

			var refPath string
			if len(m) > 1 {
				refPath = strings.TrimSpace(m[1])
			}
			// baseURL を渡して絶対パスを解決するのだ
			fullPath := resolveFullPath(baseURL, refPath)

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
					// SpeakerIDはシステム内で一意に扱うため、小文字に正規化する
					currentPage.SpeakerID = strings.ToLower(val)
				case fieldKeyText:
					currentPage.Dialogue = val
				case fieldKeyAction:
					currentPage.VisualAnchor = val
				case fieldKeyLayout:
					// 予約済み
				default:
					slog.Debug("Markdown内に未知のフィールドキーが見つかりました", "key", key)
				}
			}
		}
	}

	// 最後のパネルの追加
	if currentPage != nil && hasContent(currentPage) {
		manga.Pages = append(manga.Pages, *currentPage)
	}

	if len(manga.Pages) == 0 {
		return nil, fmt.Errorf("有効なパネル情報が見つかりませんでした")
	}

	return manga, nil
}

// resolveFullPath はベースURLと相対パスから絶対URLを構築するのだ。
func resolveFullPath(baseURL string, refPath string) string {
	if refPath == "" {
		return ""
	}

	// URLをパースし、SchemeとHostが存在すれば絶対URLとみなす
	u, err := url.Parse(refPath)
	if err == nil && u.Scheme != "" && u.Host != "" {
		return refPath
	}

	return baseURL + refPath
}

// hasContent はパネルに有効な情報が含まれているか判定します。
func hasContent(page *domain.MangaPage) bool {
	return page.ReferenceURL != "" || page.Dialogue != "" || page.VisualAnchor != ""
}
