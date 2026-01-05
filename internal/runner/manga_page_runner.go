package runner

import (
	"context"
	"fmt"

	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-manga-kit/pkg/parser"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
)

// PageRunner は MarkdownのパースとPipelineの実行を管理するのだ。
type PageRunner interface {
	Run(ctx context.Context, markdownContent string) ([]*imagedom.ImageResponse, error)
}

// MangaPageRunner MarkdownのパースとPipelineの実行を管理するのだ。
type MangaPageRunner struct {
	pageGen        *generator.PageGenerator
	markdownParser *parser.Parser
}

// NewMangaPageRunner マンガ生成パイプライン、共通スタイルサフィックス、Markdownパーサーを依存性として設定します。
func NewMangaPageRunner(mangaGen generator.MangaGenerator, styleSuffix string, markdownParser *parser.Parser) *MangaPageRunner {
	return &MangaPageRunner{
		pageGen:        generator.NewPageGenerator(mangaGen, styleSuffix),
		markdownParser: markdownParser,
	}
}

// Run 提供されたMarkdownコンテンツを処理し、設定済みのパイプラインを使用して
// 複数枚のマンガページ画像を生成するのだ（チャンク対応版）。
func (r *MangaPageRunner) Run(ctx context.Context, markdownContent string) ([]*imagedom.ImageResponse, error) {
	manga, err := r.markdownParser.Parse(markdownContent)
	if err != nil {
		return nil, fmt.Errorf("Markdownコンテンツのパースに失敗しました: %w", err)
	}
	if manga == nil {
		return nil, fmt.Errorf("エラーなしでマンガのパース結果が nil になりました。")
	}

	return r.pageGen.ExecuteMangaPages(ctx, *manga)
}
