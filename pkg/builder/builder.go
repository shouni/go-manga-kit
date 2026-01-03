package builder

import (
	"github.com/shouni/go-manga-kit/pkg/director"
	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-manga-kit/pkg/parser"
	"github.com/shouni/go-manga-kit/pkg/pipeline"
	"github.com/shouni/go-manga-kit/pkg/publisher"
	"github.com/shouni/go-text-format/pkg/md2html"
)

// MangaKit はライブラリの全機能を統合したエントリーポイントなのだ。
type MangaKit struct {
	Pipeline  *pipeline.Pipeline
	Publisher *publisher.WebtoonPublisher
	Layout    *director.LayoutManager
	Style     *director.StyleManager
}

// BuildDefaultPipeline は、標準的な設定でパイプラインを構築するのだ。
// scriptURL は、台本内の相対パス画像を解決するためのベースURLなのだよ。
func BuildDefaultPipeline(adapter pipeline.ImageGenerator, scriptURL string, styleSuffix string) (*pipeline.Pipeline, error) {
	// 1. パーサーの初期化
	// parser.NewParser(scriptURL) とすることで、引数不足を解消したのだ！
	p := parser.NewParser(scriptURL)

	// 2. プロンプトビルダーの初期化
	b := generator.NewPromptBuilder(make(generator.DNAMap), styleSuffix)

	// 3. パイプラインの構築
	return pipeline.NewPipeline(p, b, adapter), nil
}

// NewMangaKit は、すべてのコンポーネントを統合して提供する便利な関数なのだ。
func NewMangaKit(adapter pipeline.ImageGenerator, conv md2html.Converter, scriptURL string, styleSuffix string) (*MangaKit, error) {
	// BuildDefaultPipeline に scriptURL を渡すように修正したのだ
	pipe, err := BuildDefaultPipeline(adapter, scriptURL, styleSuffix)
	if err != nil {
		return nil, err
	}

	pub, err := publisher.NewWebtoonPublisher(conv)
	if err != nil {
		return nil, err
	}

	return &MangaKit{
		Pipeline:  pipe,
		Publisher: pub,
		Layout:    director.NewLayoutManager(),
		Style:     director.NewStyleManager(),
	}, nil
}
