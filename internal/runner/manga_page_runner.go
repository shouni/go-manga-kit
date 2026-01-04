package runner

import (
	"context"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	mangakit "github.com/shouni/go-manga-kit/pkg/pipeline"
)

// PageRunner は MarkdownのパースとPipelineの実行を管理するのだ。
type PageRunner interface {
	Run(ctx context.Context, markdownContent string) (*imagedom.ImageResponse, error)
}

// MangaPageRunner MarkdownのパースとPipelineの実行を管理するのだ。
type MangaPageRunner struct {
	pipeline  *mangakit.PagePipeline
	scriptURL string
}

// NewMangaPageRunner MangaPageRunnerを初期化。マンガ生成パイプライン、共通スタイル、入力ソースURLを設定する
func NewMangaPageRunner(mangaPipeline mangakit.Pipeline, styleSuffix string, scriptURL string) *MangaPageRunner {
	return &MangaPageRunner{
		pipeline:  mangakit.NewPagePipeline(mangaPipeline, styleSuffix), // mangaPipeline全体を渡す
		scriptURL: scriptURL,
	}
}

// Run 提供されたMarkdownコンテンツを処理し、設定済みのパイプラインを使用してマンガのページ画像を生成する
func (r *MangaPageRunner) Run(ctx context.Context, markdownContent string) (*imagedom.ImageResponse, error) {
	return r.pipeline.ExecuteMangaPage(ctx, markdownContent, r.scriptURL)
}
