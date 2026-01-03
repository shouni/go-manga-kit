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
	// path.Dir は、パスがファイル名のみ（"script.md"）の場合に "." を、
	// ルートディレクトリ（"/script.md"）の場合に "/" を返します。
	// これらを空文字列に正規化することで、ベースURLを正しく構築できるようにします。
	if dir == "." || dir == "/" {
		dir = ""
	}

	switch u.Scheme {
	case "gs":
		// GCSの場合は Google Cloud Storage の公開URL形式に変換します
		baseURL := &url.URL{
			Scheme: "https",
			Host:   "storage.googleapis.com",
		}

		// バケット名(u.Host)とディレクトリパス(dir)を安全に結合します
		// JoinPathは空の要素を無視するため、正規化されたdirが空でも問題ありません
		finalURL := baseURL.JoinPath(u.Host, strings.TrimPrefix(dir, "/"))

		// 構造体の Path フィールドを直接操作し、ディレクトリであることを示す
		// スラッシュを末尾に保証します。これによりクエリ等が含まれてもURLが破損しません。
		if !strings.HasSuffix(finalURL.Path, "/") {
			finalURL.Path += "/"
		}
		return finalURL.String()

	case "http", "https":
		// ベースURLにクエリパラメータやフラグメントは不要なため、
		// 必要な要素のみで新しいURLオブジェクトを構築する。
		baseURL := &url.URL{
			Scheme: u.Scheme,
			Host:   u.Host,
			Path:   dir,
		}
		if !strings.HasSuffix(baseURL.Path, "/") {
			baseURL.Path += "/"
		}
		return baseURL.String()

	default:
		slog.Debug("未対応のURLスキームです。ベースURLの解決をスキップします", "scheme", u.Scheme)
		return ""
	}
}
