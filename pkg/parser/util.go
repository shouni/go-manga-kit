package parser

import (
	"fmt"
	"net/url"
	"path"
)

func resolveBaseURL(scriptURL string) string {
	if scriptURL == "" {
		return ""
	}
	u, err := url.Parse(scriptURL)
	if err != nil {
		return ""
	}

	if u.Scheme == "gs" {
		dirPath := path.Dir(u.Path)
		// 末尾にスラッシュをつけて、後の結合を楽にするのだ
		return fmt.Sprintf("https://storage.googleapis.com/%s%s/", u.Host, dirPath)
	}

	// http/https の場合もディレクトリ階層をベースURLにするのだ
	if u.Scheme == "http" || u.Scheme == "https" {
		u.Path = path.Dir(u.Path)
		return u.String() + "/"
	}

	return ""
}
