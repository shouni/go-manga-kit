package parser

import (
	"log/slog"
	"net/url"
	"path"
)

// resolveBaseURL はスクリプトのURLからアセット参照用のベースURLを導き出すのだ
func resolveBaseURL(scriptURL string) string {
	if scriptURL == "" {
		return ""
	}

	u, err := url.Parse(scriptURL)
	if err != nil {
		// slog を使って構造化された警告ログを出すのだ
		slog.Warn("scriptURLの解析に失敗したのだ",
			"url", scriptURL,
			"error", err,
		)
		return ""
	}

	dir := path.Dir(u.Path)
	// ルートディレクトリの場合の挙動を正規化するのだ
	if dir == "." || dir == "/" {
		dir = ""
	}

	switch u.Scheme {
	case "gs":
		// GCSの場合は Google Cloud Storage の公開URL形式に変換するのだ
		// url.URL 構造体を使って安全に組み立てるのがプロの技なのだ
		baseURL := &url.URL{
			Scheme: "https",
			Host:   "storage.googleapis.com",
			Path:   path.Join(u.Host, dir) + "/",
		}
		return baseURL.String()

	case "http", "https":
		// HTTP/S の場合はパスをディレクトリ階層までに止めて末尾にスラッシュをつけるのだ
		u.Path = dir + "/"
		return u.String()

	default:
		slog.Debug("未対応のURLスキームなのだ。ベースURLの解決をスキップするのだ", "scheme", u.Scheme)
		return ""
	}
}
