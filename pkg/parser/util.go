package parser

import (
	"log/slog"
	"net/url"
	"path"
	"strings"
)

// resolveBaseURL はスクリプトのURLからアセット参照用のベースURLを導出します。
func resolveBaseURL(scriptURL string) string {
	if scriptURL == "" {
		return ""
	}

	u, err := url.Parse(scriptURL)
	if err != nil {
		slog.Warn("scriptURLの解析に失敗しました",
			"url", scriptURL,
			"error", err,
		)
		return ""
	}

	dir := path.Dir(u.Path)
	// ルートディレクトリの場合の挙動を正規化します
	if dir == "." || dir == "/" {
		dir = ""
	}

	switch u.Scheme {
	case "gs":
		// GCSの場合は Google Cloud Storage の公開URL形式に変換します
		// url.URL 構造体を使用して安全に組み立てます
		baseURL := &url.URL{
			Scheme: "https",
			Host:   "storage.googleapis.com",
		}
		// u.Host (バケット名) と dir (ディレクトリパス) をパス要素として安全に結合します
		pathElements := []string{u.Host}
		if trimmedDir := strings.TrimPrefix(dir, "/"); trimmedDir != "" {
			pathElements = append(pathElements, trimmedDir)
		}
		finalURL := baseURL.JoinPath(pathElements...)

		// ディレクトリを示すために末尾にスラッシュを追加
		return finalURL.String() + "/"

	case "http", "https":
		// HTTP/S の場合はパスをディレクトリ階層までとし、末尾にスラッシュを付与します
		u.Path = dir + "/"
		return u.String()

	default:
		slog.Debug("未対応のURLスキームです。ベースURLの解決をスキップします", "scheme", u.Scheme)
		return ""
	}
}
