package asset

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/shouni/go-remote-io/pkg/remoteio"
)

// DefaultPageFileName は共通のベースファイル名
const DefaultPageFileName = "manga_page.png"

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

// ResolveBaseURL は path からディレクトリ部分（ベースURL）を安全に抽出します。
func ResolveBaseURL(rawPath string) string {
	if rawPath == "" {
		return ""
	}

	u, err := url.Parse(rawPath)
	if err == nil && u.IsAbs() {
		dotRef, _ := url.Parse(".")
		baseURL := u.ResolveReference(dotRef).String()

		if !strings.HasSuffix(baseURL, "/") {
			baseURL += "/"
		}
		return baseURL
	}

	baseDir := filepath.Dir(rawPath)
	if baseDir == "." {
		return "./"
	}

	if !strings.HasSuffix(baseDir, string(filepath.Separator)) {
		baseDir += string(filepath.Separator)
	}

	return baseDir
}

// GenerateIndexedPath は、指定されたベースパスの拡張子の前に連番を挿入し、
// 新しいパス文字列を生成します。index は1以上の整数である必要があります。
// 例: "path/to/image.png", 1 -> "path/to/image_1.png"
func GenerateIndexedPath(basePath string, index int) (string, error) {
	if index <= 0 {
		return "", fmt.Errorf("index must be a positive integer, but got %d", index)
	}
	ext := filepath.Ext(basePath)
	base := strings.TrimSuffix(basePath, ext)
	return fmt.Sprintf("%s_%d%s", base, index, ext), nil
}
