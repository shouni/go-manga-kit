package pipeline

import (
	"context"
	"fmt"

	"github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/generator"
)

// Parser は台本テキストを構造化データに変換するインターフェースです。
type Parser interface {
	Parse(input string) (*domain.MangaResponse, error)
}

// PromptBuilder はAIへの指示文を構築するインターフェースです。
type PromptBuilder interface {
	BuildUnifiedPrompt(page domain.MangaPage) (string, int32)
	BuildFullPagePrompt(title string, panels []string, dnas []generator.CharacterDNA) string
}

// ImageGenerator は画像生成エンジン（アダプター）へのインターフェースです。
type ImageGenerator interface {
	GenerateMangaPage(ctx context.Context, req domain.ImagePageRequest) (*domain.ImageResponse, error)
}

// Pipeline はマンガ生成の全工程をオーケストレートする司令塔です。
type Pipeline struct {
	parser    Parser
	builder   PromptBuilder
	generator ImageGenerator
}

// NewPipeline は各コンポーネントのインターフェースを受け取り、Pipeline インスタンスを生成します。
func NewPipeline(p Parser, b PromptBuilder, g ImageGenerator) *Pipeline {
	return &Pipeline{
		parser:    p,
		builder:   b,
		generator: g,
	}
}

// Execute は台本の解析から画像生成までを一気通貫で実行します。
func (pl *Pipeline) Execute(ctx context.Context, markdown string, dnas []generator.CharacterDNA) (*domain.ImageResponse, error) {
	// 1. 台本の解析
	manga, err := pl.parser.Parse(markdown)
	if err != nil {
		return nil, fmt.Errorf("pipeline: 台本の解析に失敗しました: %w", err)
	}

	// 2. パネルごとのプロンプト構築とリファレンスURLの収集
	panelPrompts := make([]string, 0, len(manga.Pages))
	refURLs := make([]string, 0)
	urlMap := make(map[string]struct{})
	var baseSeed *int32

	for i, page := range manga.Pages {
		// 個別プロンプトの生成
		prompt, seed := pl.builder.BuildUnifiedPrompt(page)
		panelPrompts = append(panelPrompts, prompt)

		// 最初のパネルのシードをベースシードとして採用
		if i == 0 && seed != 0 {
			s := seed
			baseSeed = &s
		}

		// リファレンスURL（構図・背景用）の収集
		if page.ReferenceURL != "" {
			if _, exists := urlMap[page.ReferenceURL]; !exists {
				urlMap[page.ReferenceURL] = struct{}{}
				refURLs = append(refURLs, page.ReferenceURL)
			}
		}
	}

	// 3. ページ全体の統合プロンプト構築
	fullPrompt := pl.builder.BuildFullPagePrompt(manga.Title, panelPrompts, dnas)

	// 4. 画像生成の実行
	// ネガティブプロンプトは、不自然な描画を防ぐための標準セットを適用
	negativePrompt := "deformed faces, mismatched eyes, cross-eyed, low-quality faces, blurry facial features, melting faces, extra limbs, merged panels, messy lineart, distorted anatomy"

	req := domain.ImagePageRequest{
		Prompt:         fullPrompt,
		NegativePrompt: negativePrompt,
		AspectRatio:    "3:4",
		Seed:           baseSeed,
		ReferenceURLs:  refURLs,
	}

	return pl.generator.GenerateMangaPage(ctx, req)
}
