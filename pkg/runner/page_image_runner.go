package runner

import (
	"context"
	"fmt"
	"log/slog"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-manga-kit/pkg/parser"
)

// MangaPageRunner は Markdownのパースと複数ページの生成（チャンク処理）を管理するのだ。
type MangaPageRunner struct {
	cfg      config.Config
	mkParser parser.Parser
	pageGen  *generator.PageGenerator
}

// NewMangaPageRunner は、設定、パーサー、および生成エンジンを依存性として注入し、MangaPageRunnerを初期化します。
func NewMangaPageRunner(cfg config.Config, mkParser parser.Parser, mangaGen generator.MangaGenerator) *MangaPageRunner {
	return &MangaPageRunner{
		cfg:      cfg,
		mkParser: mkParser,
		pageGen:  generator.NewPageGenerator(mangaGen, cfg.StyleSuffix),
	}
}

// Run は提供されたassetPathのMarkdownコンテンツを解析し、複数枚の漫画ページ画像を生成します。
func (r *MangaPageRunner) Run(ctx context.Context, markdownAssetPath string) ([]*imagedom.ImageResponse, error) {
	manga, err := r.mkParser.ParseFromPath(ctx, markdownAssetPath)
	if err != nil {
		return nil, fmt.Errorf("markdownコンテンツのパースに失敗しました: %w", err)
	}
	if manga == nil {
		return nil, fmt.Errorf("マンガのパース結果が nil になりました")
	}

	slog.Info("Runner内部でのパース結果確認",
		"title", manga.Title,
		"panel_count", len(manga.Pages),
	)

	// ページ生成エンジンを実行して、画像バイナリ群を取得
	return r.pageGen.ExecuteMangaPages(ctx, *manga)
}
