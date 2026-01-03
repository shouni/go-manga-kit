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
			// u.Host (バケット名) と dir (ディレクトリパス) を結合します
			// path.Join で要素が欠落しないよう、dir の先頭スラッシュを除去してから結合します
			Path: path.Join(u.Host, strings.TrimPrefix(dir, "/")) + "/",
		}
		return baseURL.String()

	case "http", "https":
		// HTTP/S の場合はパスをディレクトリ階層までとし、末尾にスラッシュを付与します
		u.Path = dir + "/"
		return u.String()

	default:
		slog.Debug("未対応のURLスキームです。ベースURLの解決をスキップします", "scheme", u.Scheme)
		return ""
	}
}
