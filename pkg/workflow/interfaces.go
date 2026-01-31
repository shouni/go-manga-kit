package workflow

import (
	"context"

	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/publisher"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
)

// Workflow は、漫画生成ワークフローの各工程を担当するRunnerを構築するためのインターフェースを定義します。
type Workflow interface {
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
	Run(ctx context.Context, scriptURL string, mode string) (*domain.MangaResponse, error)
}

// PanelImageRunner は、解析済みの漫画データと対象パネルのインデックスを基に、パネル画像を生成する責務を持ちます。
type PanelImageRunner interface {
	Run(ctx context.Context, manga *domain.MangaResponse) ([]*imagedom.ImageResponse, error)
	RunAndSave(ctx context.Context, manga *domain.MangaResponse, scriptPath string) (*domain.MangaResponse, error)
}

// PageImageRunner は、解析済みの漫画データから漫画のページ画像を生成する責務を持ちます。
type PageImageRunner interface {
	Run(ctx context.Context, manga *domain.MangaResponse) ([]*imagedom.ImageResponse, error)
	RunAndSave(ctx context.Context, manga *domain.MangaResponse, plotPath string) ([]string, error)
}

// PublishRunner は、漫画データを統合し、指定された形式（例: HTML）で出力する責務を持ちます。
type PublishRunner interface {
	Run(ctx context.Context, manga *domain.MangaResponse, outputDir string) (publisher.PublishResult, error)
	BuildMarkdown(manga *domain.MangaResponse) string
}
