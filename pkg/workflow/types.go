package workflow

import (
	"context"
	"time"

	// package imagedom は画像生成サービスに関連するドメインモデルを扱います。
	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	// package mangadom は漫画制作キット自体のドメインモデルを扱います。
	mangadom "github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/publisher"
)

const (
	defaultCacheExpiration = 5 * time.Minute
	cacheCleanupInterval   = 15 * time.Minute
	defaultTTL             = 5 * time.Minute
)

// WorkflowBuilder は、さまざまな漫画処理ランナー（実行環境）を構築するためのビルダー・インターフェースを定義します。
type WorkflowBuilder interface {
	BuildDesignRunner() (DesignRunner, error)
	BuildScriptRunner() (ScriptRunner, error)
	BuildPanelImageRunner() (PanelImageRunner, error)
	BuildPageImageRunner() (PageImageRunner, error)
	BuildPublishRunner() (PublishRunner, error)
}

// DesignRunner は、キャラクターIDに基づいてデザインシートを生成し、Seed値を特定する責務を持ちます。
type DesignRunner interface {
	Run(ctx context.Context, charIDs []string, seed int64, outputDir string) (string, int64, error)
}

// ScriptRunner は、ソース（URLやテキスト）を解析し、構造化された漫画台本を生成する責務を持ちます。
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
	Run(ctx context.Context, markdownPath string) ([]*imagedom.ImageResponse, error)
	RunAndSave(ctx context.Context, markdownPath string, explicitOutputDir string) ([]string, error)
}
