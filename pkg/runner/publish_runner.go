package runner

import (
	"context"

	"github.com/shouni/go-manga-kit/pkg/workflow"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/publisher"
)

// DefaultPublisherRunner は pkg/publisher を利用した標準実装なのだ。
type DefaultPublisherRunner struct {
	cfg       workflow.Config
	publisher *publisher.MangaPublisher
}

func NewDefaultPublisherRunner(cfg workflow.Config, pub *publisher.MangaPublisher) *DefaultPublisherRunner {
	return &DefaultPublisherRunner{
		cfg:       cfg,
		publisher: pub,
	}
}

func (pr *DefaultPublisherRunner) Run(ctx context.Context, manga domain.MangaResponse, images []*imagedom.ImageResponse, outputDir string) error {
	opts := publisher.Options{
		OutputImageDir: outputDir,
		ImageDirName:   "images",
	}

	return pr.publisher.Publish(ctx, manga, images, opts)
}
