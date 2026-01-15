package asset

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shouni/go-utils/urlpath"
)

const (
	// DefaultImageDir は生成された画像を格納するデフォルトのディレクトリ名です。
	DefaultImageDir = "images"
	// DefaultMangaPlotName は生成された漫画プロットのデフォルトファイル名です。
	DefaultMangaPlotName = "manga_plot.md"
	// DefaultPanelFileName はパネル画像の共通のベースファイル名です。
	DefaultPanelFileName = "panel.png"
	// DefaultPageFileName はページ画像の共通のベースファイル名です。
	DefaultPageFileName = "manga_page.png"
)

var (
	// PanelFileRegex はパネル画像 (panel_1.png 等) に一致します
	PanelFileRegex = createIndexedRegex(DefaultPanelFileName)
	// PageFileRegex はページ画像 (manga_page_1.png 等) に一致します
	PageFileRegex = createIndexedRegex(DefaultPageFileName)
)

// ResolveOutputPath は、ベースとなるディレクトリパスとファイル名から、
// GCS/ローカルを考慮した最終的な出力パスを生成します。
func ResolveOutputPath(baseDir, fileName string) (string, error) {
	return urlpath.ResolveOutputPath(baseDir, fileName)
}

// ResolveBaseURL は、入力パス（URLまたはローカルパス）から
// 親ディレクトリのパスを解決し、末尾がセパレータで終わるように正規化します。
func ResolveBaseURL(rawPath string) string {
	return urlpath.ResolveBaseURL(rawPath)
}

// GenerateIndexedPath は、指定されたベースパスの拡張子の前に連番を挿入し、
// 新しいパス文字列を生成します。index は1以上の整数である必要があります。
// 例: "path/to/image.png", 1 -> "path/to/image_1.png"
func GenerateIndexedPath(basePath string, index int) (string, error) {
	return urlpath.GenerateIndexedPath(basePath, index)
}

// 正規表現生成を共通化するためのヘルパー関数（非公開）
func createIndexedRegex(fileName string) *regexp.Regexp {
	baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	pattern := fmt.Sprintf(`^%s_\d+\.png$`, regexp.QuoteMeta(baseName))
	return regexp.MustCompile(pattern)
}
