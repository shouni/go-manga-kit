package asset

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

// ResolveBaseURL は、入力パス（URLまたはローカルパス）から
// 親ディレクトリのパスを解決し、末尾がセパレータで終わるように正規化します。
func ResolveBaseURL(rawPath string) string {
	if rawPath == "" {
		return ""
	}

	u, err := url.Parse(rawPath)
	if err == nil && u.IsAbs() {
		// URL形式の場合、"." への参照を解決することで親ディレクトリのURLを取得
		dotRef, _ := url.Parse(".")
		baseURL := u.ResolveReference(dotRef).String()

		// ディレクトリパスであることを保証するため、末尾に "/" を追加
		if !strings.HasSuffix(baseURL, "/") {
			baseURL += "/"
		}
		return baseURL
	}

	// URLスキームがない場合はローカルファイルパスとして扱う
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
