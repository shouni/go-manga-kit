package publisher

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// ResolveOutputPath は、ベースとなるディレクトリパスとファイル名から、
// GCS/ローカルを考慮した最終的な出力パスを生成します。
func ResolveOutputPath(baseDir, fileName string) (string, error) {
	if strings.HasPrefix(strings.ToLower(baseDir), "gs://") {
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

// ResolveBaseURL は path からディレクトリ部分（ベースURL）を安全に抽出します。
func ResolveBaseURL(rawPath string) string {
	if rawPath == "" {
		return ""
	}

	// 1. スキーム（gs://, http://等）があるか確認するのだ
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
