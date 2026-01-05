package runner

import (
	"context"

	"github.com/shouni/go-manga-kit/pkg/domain"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/internal/config"
	"github.com/shouni/go-manga-kit/pkg/publisher"
)

// PublisherRunner はパブリッシュ処理のインターフェースです。
type PublisherRunner interface {
	Run(ctx context.Context, manga domain.MangaResponse, images []*imagedom.ImageResponse) error
}

// DefaultPublisherRunner は pkg/publisher を利用した標準実装です。
type DefaultPublisherRunner struct {
	options   config.GenerateOptions
	publisher *publisher.MangaPublisher
}

func NewDefaultPublisherRunner(options config.GenerateOptions, pub *publisher.MangaPublisher) *DefaultPublisherRunner {
	return &DefaultPublisherRunner{
		options:   options,
		publisher: pub,
	}
}

func (pr *DefaultPublisherRunner) Run(ctx context.Context, manga domain.MangaResponse, images []*imagedom.ImageResponse) error {
	// internal/config の値を pkg/publisher 用の構造体に詰め替えます。
	opts := publisher.Options{
		OutputFile:     pr.options.OutputFile,
		OutputImageDir: pr.options.OutputImageDir,
		ImageDirName:   "images",
	}

	return pr.publisher.Publish(ctx, manga, images, opts)
}
