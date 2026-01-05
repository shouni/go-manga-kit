package runner

import (
	"context"
	"log/slog"

	"github.com/shouni/go-manga-kit/internal/config"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/generator"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
)

// ImageRunner は、漫画の台本データを基に画像を生成するためのインターフェース。
type ImageRunner interface {
	// Run は台本の全ページに対して画像生成を実行し、結果のリストを返します
	Run(ctx context.Context, manga domain.MangaResponse) ([]*imagedom.ImageResponse, error)
}

// MangaImageRunner は、アプリケーション層の実行管理を担う実体なのだ。
type MangaImageRunner struct {
	groupGen *generator.GroupGenerator // 汎用化された生成GroupGenerator
	limit    int                       // 生成パネル数の制限（テスト用）
}

// NewMangaImageRunner は、依存関係を注入して Runner を初期化します
func NewMangaImageRunner(mangaGen generator.MangaGenerator, styleSuffix string, limit int) *MangaImageRunner {

	// pkg/generator にある汎用Generatorを構築します
	groupGen := generator.NewGroupGenerator(mangaGen, styleSuffix, config.DefaultRateLimit)

	return &MangaImageRunner{
		groupGen: groupGen,
		limit:    limit,
	}
}

// Run は、設定された制限やログ出力を管理しながら、Generatorを実行します
func (ir *MangaImageRunner) Run(ctx context.Context, manga domain.MangaResponse) ([]*imagedom.ImageResponse, error) {
	pages := manga.Pages

	// 1. パネル数制限の適用
	if ir.limit > 0 && len(pages) > ir.limit {
		slog.Info("パネル数に制限を適用したのだ", "limit", ir.limit, "total", len(pages))
		pages = pages[:ir.limit]
	}

	slog.Info("並列画像生成を開始するのだ",
		"count", len(pages),
		"interval", config.DefaultRateLimit,
	)

	// 2. 汎用パイプラインへの処理委譲
	images, err := ir.groupGen.ExecutePanelGroup(ctx, pages)
	if err != nil {
		slog.Error("画像生成パイプラインの実行中にエラーが発生したのだ", "error", err)
		return nil, err
	}

	slog.Info("すべてのパネルが正常に生成されたのだ", "total", len(images))
	return images, nil
}
