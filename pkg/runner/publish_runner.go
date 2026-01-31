package runner

import (
	"context"

	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/publisher"
)

// DefaultPublisherRunner は pkg/publisher を利用した標準実装なのだ。
type DefaultPublisherRunner struct {
	cfg       config.Config
	publisher *publisher.MangaPublisher
}

func NewDefaultPublisherRunner(cfg config.Config, pub *publisher.MangaPublisher) *DefaultPublisherRunner {
	return &DefaultPublisherRunner{
		cfg:       cfg,
		publisher: pub,
	}
}

func (pr *DefaultPublisherRunner) Run(ctx context.Context, manga *domain.MangaResponse, outputDir string) (publisher.PublishResult, error) {
	opts := publisher.Options{
		OutputDir: outputDir,
	}

	return pr.publisher.Publish(ctx, manga, opts)
}

// BuildMarkdown は保存処理を行わず、構造体から Markdown 文字列のみを生成して返却します。
// Webハンドラーでの表示用などで、署名付きURLに置換済みのデータを扱う際に便利です。
func (pr *DefaultPublisherRunner) BuildMarkdown(manga *domain.MangaResponse) string {
	// 内部の publisher.BuildMarkdownOnly を呼び出す
	// 第2引数の imagePaths を nil にすることで、構造体内のパス（署名付きURLなど）をそのまま使用します
	return pr.publisher.BuildMarkdown(manga, nil)
}
