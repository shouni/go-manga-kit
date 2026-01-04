package runner

import (
	"context"
	"log/slog"
	"strings"

	"github.com/shouni/go-manga-kit/internal/config"
	mngdom "github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/pipeline"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/gemini-image-kit/pkg/generator"
)

// ImageRunner は、漫画の台本データを基に画像を生成するためのインターフェース。
type ImageRunner interface {
	// Run は台本の全ページに対して画像生成を実行し、結果のリストを返すのだ。
	Run(ctx context.Context, manga mngdom.MangaResponse) ([]*imagedom.ImageResponse, error)
}

// MangaImageRunner は、アプリケーション層の実行管理を担う実体なのだ。
type MangaImageRunner struct {
	pipeline   *pipeline.GroupPipeline     // 汎用化された生成パイプライン
	characters map[string]mngdom.Character // 利用可能なキャラクター設定
	limit      int                         // 生成パネル数の制限（テスト用）
}

// NewMangaImageRunner は、依存関係を注入して Runner を初期化するのだ。
func NewMangaImageRunner(imgGen generator.ImageGenerator, chars map[string]mngdom.Character, limit int, basePrompt string) *MangaImageRunner {
	// IDを小文字に統一して検索しやすくするのだ
	normalizedChars := make(map[string]mngdom.Character)
	for k, v := range chars {
		normalizedChars[strings.ToLower(k)] = v
	}

	// pkg/adapters に切り出した汎用パイプラインを構築するのだ
	pipeline := pipeline.NewGroupPipeline(imgGen, basePrompt, config.DefaultRateLimit)

	return &MangaImageRunner{
		pipeline:   pipeline,
		characters: normalizedChars,
		limit:      limit,
	}
}

// Run は、設定された制限やログ出力を管理しながら、パイプラインを実行するのだ！
func (ir *MangaImageRunner) Run(ctx context.Context, manga mngdom.MangaResponse) ([]*imagedom.ImageResponse, error) {
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
	// ここで「どう生成するか」の重たいロジックは Pipeline に任せるのだ！
	images, err := ir.pipeline.Execute(ctx, pages, ir.characters)
	if err != nil {
		slog.Error("画像生成パイプラインの実行中にエラーが発生したのだ", "error", err)
		return nil, err
	}

	slog.Info("すべてのパネルが正常に生成されたのだ", "total", len(images))
	return images, nil
}
