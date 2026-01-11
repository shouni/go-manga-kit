package runner

import (
	"context"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
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

func (pr *DefaultPublisherRunner) Run(ctx context.Context, manga domain.MangaResponse, images []*imagedom.ImageResponse, outputDir string) (publisher.PublishResult, error) {
	opts := publisher.Options{
		OutputDir: outputDir,
	}

	return pr.publisher.Publish(ctx, manga, images, opts)
}
