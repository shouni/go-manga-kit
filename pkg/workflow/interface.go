package workflow

import (
	"context"

	imgdom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/publisher"
)

// DesignRunner はキャラクターIDに基づいてデザインシートを生成し、Seed値を特定するのだ。
type DesignRunner interface {
	Run(ctx context.Context, charIDs []string, seed int64, outputDir string) (string, int64, error)
}

// ScriptRunner はソース（URLやテキスト）を解析し、構造化された漫画台本を生成するのだ。
type ScriptRunner interface {
	Run(ctx context.Context, scriptURL string, mode string) (domain.MangaResponse, error)
}

// PanelImageRunner は台本データを基に、指定されたパネルの画像を生成するのだ。
type PanelImageRunner interface {
	Run(ctx context.Context, manga domain.MangaResponse, targetIndices []int) ([]*imgdom.ImageResponse, error)
}

// PublishRunner は生成された画像と台本を統合し、HTMLやMarkdownとして保存するのだ。
type PublishRunner interface {
	Run(ctx context.Context, manga domain.MangaResponse, images []*imgdom.ImageResponse, outputDir string) (publisher.PublishResult, error)
}

// PageImageRunner は提供された fullPathからMarkdown コンテンツを解析し、複数枚の漫画ページ画像を生成するのだ。
type PageImageRunner interface {
	Run(ctx context.Context, fullPath string) ([]*imgdom.ImageResponse, error)
}
