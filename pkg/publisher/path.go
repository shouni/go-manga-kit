package publisher

import (
	"fmt"
	"net/url"
	"path"
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
		// url.JoinPath を使用して、GCS スキームを維持したままパスを安全に結合します
		newPath, err := url.JoinPath(u.String(), fileName)
		if err != nil {
			return "", fmt.Errorf("GCSパスの結合に失敗しました: %w", err)
		}
		return newPath, nil
	}
	return filepath.Join(baseDir, fileName), nil
}

// ResolveFullPath は、相対的な参照パスをベースURLと結合し、絶対URLを生成します。
func ResolveFullPath(baseURL string, refPath string) string {
	if refPath == "" {
		return ""
	}

	// 1. 既に絶対URL（スキームを持つ形式）であるかを確認します
	u, err := url.Parse(refPath)
	if err == nil && u.Scheme != "" && u.IsAbs() {
		return refPath
	}

	// 2. 相対パスをベースURLと結合し、完全なURLを生成します
	if baseURL != "" {
		// url.JoinPath はスラッシュの重複を自動的に解決し、堅牢にパスを結合します
		fullPath, err := url.JoinPath(baseURL, refPath)
		if err == nil {
			return fullPath
		}
		// JoinPath が失敗した場合は、フォールバックとして単純結合を試みます
		return strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(refPath, "/")
	}

	return refPath
}

// ResolveBaseURL は rawURL からディレクトリパスを安全に抽出します。
func ResolveBaseURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	u, err := url.Parse(rawURL)
	if err == nil && u.Scheme != "" {
		// URL のパス部分に対してのみ path.Dir を適用し、スキーマを維持したままディレクトリを特定します
		u.Path = path.Dir(u.Path)
		return u.String()
	}

	// スキーマがない場合は、単純なファイルパスとして path.Dir を適用します
	return path.Dir(rawURL)
}
