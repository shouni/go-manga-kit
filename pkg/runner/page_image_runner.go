package runner

import (
	"context"
	"fmt"

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

// NewMangaPageRunner は生成エンジン、スタイル設定、パーサーを依存性として注入して初期化するのだ。
func NewMangaPageRunner(cfg config.Config, mkParser parser.Parser, mangaGen generator.MangaGenerator, styleSuffix string) *MangaPageRunner {
	return &MangaPageRunner{
		cfg:      cfg,
		mkParser: mkParser,
		pageGen:  generator.NewPageGenerator(mangaGen, styleSuffix),
	}
}

// Run は提供された Markdown コンテンツを解析し、複数枚の漫画ページ画像を生成するのだ。
func (r *MangaPageRunner) Run(ctx context.Context, scriptURL, markdownContent string) ([]*imagedom.ImageResponse, error) {
	//markdownParser := parser.NewParser(scriptURL)
	//manga, err := markdownParser.Parse(markdownContent)
	manga, err := r.mkParser.Parse(scriptURL, markdownContent)
	if err != nil {
		return nil, fmt.Errorf("markdownコンテンツのパースに失敗しました: %w", err)
	}
	if manga == nil {
		return nil, fmt.Errorf("マンガのパース結果が nil になりました")
	}

	// ページ生成エンジンを実行して、画像バイナリ群を取得
	return r.pageGen.ExecuteMangaPages(ctx, *manga)
}
