package generator

import (
	"context"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/domain"

	"github.com/shouni/gemini-image-kit/pkg/generator"
)

const (
	// MaxPanelsPerPage は1枚の漫画ページに含めるパネルの最大数です。
	MaxPanelsPerPage = 6

	// PanelAspectRatio は単体パネル（1コマ）の推奨アスペクト比です。
	PanelAspectRatio = "16:9"

	// PageAspectRatio は統合ページ全体の推奨アスペクト比です。
	PageAspectRatio = "3:4"
)

// ImagePromptBuilder は、画像生成用のプロンプトを構築するためのインターフェースを定義します。
// これにより、漫画のパネルやページに対するプロンプト生成ロジックを抽象化します。
type ImagePromptBuilder interface {
	// BuildPanelPrompt は、単一の漫画パネル用のユーザープロンプト、システムプロンプト、
	// および再現性のためのseed値を生成します。
	BuildPanelPrompt(panel domain.Panel, speakerID string) (string, string, int64)

	// BuildMangaPagePrompt は、統合された漫画ページ画像用のユーザープロンプトと
	// システムプロンプトを生成します。
	BuildMangaPagePrompt(panels []domain.Panel, refURLs []string, mangaTitle string) (userPrompt string, systemPrompt string)
}

// PanelsImageGenerator は、指定されたコンテキスト内で一連のパネルの画像レスポンスを生成するためのインターフェースを定義します。
type PanelsImageGenerator interface {
	Execute(ctx context.Context, panels []domain.Panel) ([]*imagedom.ImageResponse, error)
}

// PagesImageGenerator は、与えられた漫画レスポンスに基づいて漫画ページの画像データを生成します。
// パネルを処理し、画像レスポンスのスライスまたは失敗時にエラーを出力します。
type PagesImageGenerator interface {
	Execute(ctx context.Context, manga domain.MangaResponse) ([]*imagedom.ImageResponse, error)
}

type MangaGenerator struct {
	ImgGen        generator.ImageGenerator
	PromptBuilder ImagePromptBuilder
	Characters    domain.CharactersMap
}
