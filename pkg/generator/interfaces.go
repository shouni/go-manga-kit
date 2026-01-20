package generator

import (
	"context"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	mangadom "github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/prompts"
)

// ImagePromptBuilder は、画像生成用のプロンプトを構築するためのインターフェースを定義します。
// これにより、漫画のパネルやページに対するプロンプト生成ロジックを抽象化します。
type ImagePromptBuilder interface {
	// BuildPanelPrompt は、単一の漫画パネル用のユーザープロンプト、システムプロンプト、および使用するseed値を決定します。
	BuildPanelPrompt(panel mangadom.Panel, speakerID string) (userPrompt string, systemPrompt string, targetSeed int64)

	// BuildMangaPagePrompt は、統合された漫画ページ画像用のユーザープロンプトと システムプロンプトを生成します。
	BuildMangaPagePrompt(panels []mangadom.Panel, rm *prompts.ResourceMap) (userPrompt string, systemPrompt string)
}

// PanelsImageGenerator は、指定されたコンテキスト内で一連のパネルの画像レスポンスを生成するためのインターフェースを定義します。
type PanelsImageGenerator interface {
	Execute(ctx context.Context, panels []mangadom.Panel) ([]*imagedom.ImageResponse, error)
}

// PagesImageGenerator は、与えられた漫画レスポンスに基づいて漫画ページの画像データを生成します。
// パネルを処理し、画像レスポンスのスライスまたは失敗時にエラーを出力します。
type PagesImageGenerator interface {
	Execute(ctx context.Context, manga *mangadom.MangaResponse) ([]*imagedom.ImageResponse, error)
}
