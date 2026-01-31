package runner

import (
	"context"

	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/publisher"
)

// MangaPublisherRunner は pkg/publisher を利用して漫画成果物の公開と構築を担います。
type MangaPublisherRunner struct {
	cfg       config.Config
	publisher *publisher.MangaPublisher
}

// NewMangaPublisherRunner は、指定された構成と MangaPublisher を持つ新しい MangaPublisherRunner インスタンスを作成します。
func NewMangaPublisherRunner(cfg config.Config, pub *publisher.MangaPublisher) *MangaPublisherRunner {
	return &MangaPublisherRunner{
		cfg:       cfg,
		publisher: pub,
	}
}

// Run は漫画データの公開処理を実行し、Markdown や HTML などの成果物を指定された出力ディレクトリに保存します。
func (pr *MangaPublisherRunner) Run(ctx context.Context, manga *domain.MangaResponse, outputDir string) (publisher.PublishResult, error) {
	opts := publisher.Options{
		OutputDir: outputDir,
	}

	return pr.publisher.Publish(ctx, manga, opts)
}

// BuildMarkdown は保存処理を行わず、構造体から Markdown 文字列のみを生成して返却します。
func (pr *MangaPublisherRunner) BuildMarkdown(manga *domain.MangaResponse) string {
	// publisher.Options を空で渡すことで、外部パス指定を行わず、
	// domain.MangaResponse 内の ReferenceURL をそのまま使用するデフォルト挙動を選択します。
	return pr.publisher.BuildMarkdown(manga, publisher.Options{})
}
