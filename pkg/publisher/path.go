package publisher

import (
	"fmt"
	"log/slog"
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
		u.Path, err = url.JoinPath(u.Path, fileName)
		if err != nil {
			return "", fmt.Errorf("GCSパスの結合に失敗しました: %w", err)
		}
		return u.String(), nil
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
	if err == nil && u.IsAbs() {
		return refPath
	}

	// 2. 相対パスをベースURLと結合し、完全なURLを生成します
	if baseURL != "" {
		base, err := url.Parse(baseURL)
		if err != nil {
			slog.Warn("無効なベースURLが渡されたため、単純結合にフォールバックします", "baseURL", baseURL, "error", err)
			return strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(refPath, "/")
		}

		ref, err := url.Parse(refPath)
		if err != nil {
			slog.Warn("無効な参照パスが渡されたため、単純結合にフォールバックします", "refPath", refPath, "error", err)
			return strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(refPath, "/")
		}

		// ResolveReference により、../ や絶対パス (/) も適切に処理されます。
		return base.ResolveReference(ref).String()
	}

	return refPath
}

// ResolveBaseURL は rawURL からディレクトリパスを安全に抽出します。
func ResolveBaseURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	u, err := url.Parse(rawURL)
	if err == nil && u.IsAbs() {
		// URL のパス部分に対してのみ path.Dir を適用し、スキーマを維持したままディレクトリを特定します
		u.Path = path.Dir(u.Path)
		return u.String()
	}

	// スキーマがない場合は、単純なファイルパスとして path.Dir を適用します
	return path.Dir(rawURL)
}
