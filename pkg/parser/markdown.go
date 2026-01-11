package parser

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/publisher"
)

const (
	fieldKeySpeaker = "speaker"
	fieldKeyText    = "text"
	fieldKeyAction  = "action"
)

// Parser は解析するためのインターフェースを定義します。
type Parser interface {
	// Parse は input（Markdown）を解析し、baseAssetURL を起点として画像パスを解決します。
	Parse(baseAssetURL string, input string) (*domain.MangaResponse, error)
}

// MarkdownParser は Markdown 形式の台本を解析する構造体です。
type MarkdownParser struct{}

// NewMarkdownParser は新しい MarkdownParser インスタンスを生成します。
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{}
}

// Parse は指定された baseAssetURL を基に参照パスを解決し、Markdown テキストを解析して
// domain.MangaResponse 構造体に変換します。
func (p *MarkdownParser) Parse(baseAssetURL string, input string) (*domain.MangaResponse, error) {
	// baseAssetURL (ファイルパス or URL) から、ディレクトリ部分を抽出するのだ。
	// これが相対パス解決の「絶対的な起点」になるのだ！
	baseURL := publisher.ResolveBaseURL(baseAssetURL)

	slog.Info("Markdownパース開始",
		"baseAssetURL", baseAssetURL,
		"resolvedAssetRoot", baseURL,
	)

	manga := &domain.MangaResponse{}
	lines := strings.Split(input, "\n")
	var currentPage *domain.MangaPage

	// 前のページを結果リストに追加するヘルパー関数
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

		// パネル解析 (## Panel: images/panel_1.png)
		if m := PanelRegex.FindStringSubmatch(trimmedLine); m != nil {
			addPreviousPage()

			var refPath string
			if len(m) > 1 {
				refPath = strings.TrimSpace(m[1])
			}

			// 起点となる baseURL と Markdown内の相対パスをガッチャンコするのだ！
			fullPath := publisher.ResolveFullPath(baseURL, refPath)

			slog.Info("パス解決の実行",
				"assetRoot", baseURL,
				"rawRef", refPath,
				"resolvedFull", fullPath,
			)

			currentPage = &domain.MangaPage{
				Page:         len(manga.Pages) + 1,
				ReferenceURL: fullPath,
			}
			continue
		}

		// フィールド行の解析 (speaker, text, action)
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

	// 最後のページを追加
	addPreviousPage()

	if len(manga.Pages) == 0 {
		return nil, fmt.Errorf("有効なパネル情報が見つかりませんでした")
	}

	return manga, nil
}

// hasContent はページが有効な情報（画像、台詞、またはアクション）を保持しているか判定します。
func hasContent(page *domain.MangaPage) bool {
	return page.ReferenceURL != "" || page.Dialogue != "" || page.VisualAnchor != ""
}
