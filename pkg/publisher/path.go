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

		// url.JoinPath はパス部分のみを安全に結合し、スキーム部分を保護します
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
			slog.Warn("無効なベースURLのため、単純結合にフォールバックします", "baseURL", baseURL, "error", err)
			return strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(refPath, "/")
		}

		// パスがスラッシュで終わっていない場合、かつ拡張子のようなドットがある場合は
		// path.Dir でディレクトリに落とし込み、末尾スラッシュを付与します。
		if !strings.HasSuffix(base.Path, "/") {
			base.Path = path.Dir(base.Path)
			if !strings.HasSuffix(base.Path, "/") {
				base.Path += "/"
			}
		}

		ref, err := url.Parse(refPath)
		if err != nil {
			slog.Warn("無効な参照パスのため、単純結合にフォールバックします", "refPath", refPath, "error", err)
			return strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(refPath, "/")
		}

		// ResolveReference により、正確に絶対URLを構築します
		resolved := base.ResolveReference(ref).String()
		slog.Debug("Path resolved", "base", baseURL, "ref", refPath, "result", resolved)
		return resolved
	}

	return refPath
}

// ResolveBaseURL は rawURL からディレクトリ部分（ベースURL）を安全に抽出します。
func ResolveBaseURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		slog.Warn("ResolveBaseURL: URLパース失敗。ファイルパスとして扱います。", "url", rawURL, "error", err)
		return path.Dir(rawURL)
	}

	if !u.IsAbs() {
		// スキーマがない場合は、単純なファイルパスとして path.Dir を適用します
		return path.Dir(rawURL)
	}

	dotRef, _ := url.Parse(".")
	return u.ResolveReference(dotRef).String()
}
