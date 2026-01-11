package workflow

import (
	"context"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	mangadom "github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/publisher"
)

// DesignRunner はキャラクターIDに基づいてデザインシートを生成し、Seed値を特定するのだ。
type DesignRunner interface {
	Run(ctx context.Context, charIDs []string, seed int64, outputDir string) (string, int64, error)
}

// ScriptRunner はソース（URLやテキスト）を解析し、構造化された漫画台本を生成するのだ。
type ScriptRunner interface {
	Run(ctx context.Context, scriptURL string, mode string) (mangadom.MangaResponse, error)
}

// PanelImageRunner は、解析済みの漫画データと対象パネルのインデックスを基に、パネル画像を生成する責務を持ちます。
type PanelImageRunner interface {
	Run(ctx context.Context, manga mangadom.MangaResponse, targetIndices []int) ([]*imagedom.ImageResponse, error)
}

// PublishRunner は、生成された画像と漫画データを統合し、指定された形式（例: HTML）で出力する責務を持ちます。
type PublishRunner interface {
	Run(ctx context.Context, manga mangadom.MangaResponse, images []*imagedom.ImageResponse, outputDir string) (publisher.PublishResult, error)
}

// PageImageRunner は、指定されたパスのMarkdownコンテンツから漫画のページ画像を生成する責務を持ちます。
type PageImageRunner interface {
	Run(ctx context.Context, assetPath string) ([]*imagedom.ImageResponse, error)
}
