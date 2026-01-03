package publisher

import (
	"context"
	"fmt"
	"path"
)

// OutputWriter はデータを外部ストレージに保存するためのインターフェースです。
type OutputWriter interface {
	Write(ctx context.Context, path string, data []byte) error
}

// AssetManager は生成物の保存パスと永続化を管理します。
type AssetManager struct {
	writer  OutputWriter
	baseDir string // 保存先のベースディレクトリ (例: "output/manga-001")
}

func NewAssetManager(writer OutputWriter, baseDir string) *AssetManager {
	return &AssetManager{
		writer:  writer,
		baseDir: baseDir,
	}
}

// SaveImage は画像データを保存し、その保存先の相対パスを返します。
func (am *AssetManager) SaveImage(ctx context.Context, fileName string, data []byte) (string, error) {
	fullPath := path.Join(am.baseDir, fileName)
	if err := am.writer.Write(ctx, fullPath, data); err != nil {
		return "", fmt.Errorf("asset_manager: 画像の保存に失敗しました: %w", err)
	}
	return fullPath, nil
}
