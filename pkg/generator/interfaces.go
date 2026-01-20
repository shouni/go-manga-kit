package generator

import (
	"context"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/domain"
)

// PanelsImageGenerator は、指定されたコンテキスト内で一連のパネルの画像レスポンスを生成するためのインターフェースを定義します。
type PanelsImageGenerator interface {
	Execute(ctx context.Context, panels []domain.Panel) ([]*imagedom.ImageResponse, error)
}

// PagesImageGenerator は、与えられた漫画レスポンスに基づいて漫画ページの画像データを生成します。
// パネルを処理し、画像レスポンスのスライスまたは失敗時にエラーを出力します。
type PagesImageGenerator interface {
	Execute(ctx context.Context, manga *domain.MangaResponse) ([]*imagedom.ImageResponse, error)
}
