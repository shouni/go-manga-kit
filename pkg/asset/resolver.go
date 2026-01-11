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

// ResolveBaseURL は path からディレクトリ部分（ベースURL）を安全に抽出します。
func ResolveBaseURL(rawPath string) string {
	if rawPath == "" {
		return ""
	}

	u, err := url.Parse(rawPath)
	if err == nil && u.IsAbs() {
		// URL形式の場合は ResolveReference(".") を使って親ディレクトリを取得するのだ
		dotRef, _ := url.Parse(".")
		baseURL := u.ResolveReference(dotRef).String()

		// ディレクトリであることを保証するために末尾を "/" にするのだ
		if !strings.HasSuffix(baseURL, "/") {
			baseURL += "/"
		}
		return baseURL
	}

	// 2. スキームがない場合は、ローカルのファイルパスとして扱うのだ
	// filepath.Dir を使ってディレクトリ部分を取り出すのだ
	baseDir := filepath.Dir(rawPath)

	// Windows環境などで "." (カレントディレクトリ) が返ってきた場合や
	// スラッシュで終わっていない場合は、末尾を整えるのだ
	if baseDir == "." {
		return "./"
	}

	// ローカルパスの区切り文字（OS依存）を末尾に付与するのだ
	if !strings.HasSuffix(baseDir, string(filepath.Separator)) {
		baseDir += string(filepath.Separator)
	}

	return baseDir
}
