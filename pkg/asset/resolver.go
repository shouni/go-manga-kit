package asset

import (
	"github.com/shouni/go-utils/urlpath"
)

// ResolveOutputPath は、ベースとなるディレクトリパスとファイル名から、
// GCS/ローカルを考慮した最終的な出力パスを生成します。
func ResolveOutputPath(baseDir, fileName string) (string, error) {
	return urlpath.ResolveOutputPath(baseDir, fileName)
}

// ResolveBaseURL は、入力パス（URLまたはローカルパス）から
// 親ディレクトリのパスを解決し、末尾がセパレータで終わるように正規化します。
func ResolveBaseURL(rawPath string) string {
	return urlpath.ResolveBaseURL(rawPath)
}

// GenerateIndexedPath は、指定されたベースパスの拡張子の前に連番を挿入し、
// 新しいパス文字列を生成します。index は1以上の整数である必要があります。
// 例: "path/to/image.png", 1 -> "path/to/image_1.png"
func GenerateIndexedPath(basePath string, index int) (string, error) {
	return urlpath.GenerateIndexedPath(basePath, index)
}
