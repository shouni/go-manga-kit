package publisher

import (
	"fmt"
	"log/slog"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

// ResolveOutputPath は、ベースとなるディレクトリパスとファイル名から、
// GCS/ローカルを考慮した最終的な出力パスを生成します。
func ResolveOutputPath(baseDir, fileName string) (string, error) {
	if strings.HasPrefix(baseDir, "gs://") {
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

// ResolveFullPath は、相対的な参照パスをベースパス（ディレクトリ）と結合し、フルパス（絶対URL/パス）を生成します。
func ResolveFullPath(baseDir string, refPath string) string {
	if refPath == "" {
		return ""
	}

	// 1. 既に絶対URL（スキームを持つ形式、例: gs:// や http://）であるかを確認するのだ
	u, err := url.Parse(refPath)
	if err == nil && u.IsAbs() {
		return refPath
	}

	// 2. ベースディレクトリが空なら、そのまま返すしかないのだ
	if baseDir == "" {
		return refPath
	}

	// 3. ベースパスを解析するのだ
	base, err := url.Parse(baseDir)
	if err != nil {
		slog.Warn("無効なベースパスのため、単純結合にフォールバックします", "baseDir", baseDir, "error", err)
		return strings.TrimSuffix(baseDir, "/") + "/" + strings.TrimPrefix(refPath, "/")
	}

	// 末尾がスラッシュで終わるように確実に調整するのだ。
	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}

	ref, err := url.Parse(refPath)
	if err != nil {
		slog.Warn("無効な参照パスのため、単純結合にフォールバックします", "refPath", refPath, "error", err)
		return strings.TrimSuffix(baseDir, "/") + "/" + strings.TrimPrefix(refPath, "/")
	}

	// ResolveReference により、正確に絶対パスを構築するのだ！
	resolved := base.ResolveReference(ref).String()
	slog.Debug("Path resolved", "base", baseDir, "ref", refPath, "result", resolved)
	return resolved
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
