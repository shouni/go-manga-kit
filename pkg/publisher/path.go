package publisher

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/shouni/go-remote-io/pkg/remoteio"
)

// ResolveOutputPath は、ベースとなるディレクトリパスとファイル名から、
// GCS/ローカルを考慮した最終的な出力パスを生成します。
func ResolveOutputPath(baseDir, fileName string) (string, error) {
	if remoteio.IsGCSURI(baseDir) {
		u, err := url.Parse(baseDir)
		if err != nil {
			return "", fmt.Errorf("無効なGCS URIです: %w", err)
		}
		u.Path, err = url.JoinPath(u.Path, fileName)
		if err != nil {
			return "", fmt.Errorf("GCSパスの結合に失敗しました: %w", err)
		}
		return u.String(), nil
	}
	return filepath.Join(baseDir, fileName), nil
}

// ResolveFullPath は、絶対URLの作成
func ResolveFullPath(baseURL string, refPath string) string {
	if refPath == "" {
		return ""
	}

	// 1. 既に完全なURL（http, https, gs）ならそのまま返す
	if strings.HasPrefix(refPath, "http://") ||
		strings.HasPrefix(refPath, "https://") ||
		strings.HasPrefix(refPath, "gs://") {
		return refPath
	}

	// 2. 相対パス（images/など）を、ベースURL（gs://.../）と結合する
	// これにより、相対パスが「gs://bucket/path/images/panel_1.png」に昇格するのだ！
	if baseURL != "" {
		// strings.TrimPrefix を使って、スラッシュの重複を防ぐのがコツなのだ
		return strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(refPath, "/")
	}

	return refPath
}
