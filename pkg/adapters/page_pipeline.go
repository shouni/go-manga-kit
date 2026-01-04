package adapters

import (
	"context"
	"strings"

	imgdomain "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/domain"
)

// PagePipeline は、マンガの1ページを統合して生成する汎用パイプラインなのだ。
type PagePipeline struct {
	builder PromptBuilder
	adapter MangaPageAdapter
	// Runnerから移管：ベースとなるURL（GCS等の変換済みURL）
	baseURL string
}

func NewPagePipeline(b PromptBuilder, a MangaPageAdapter, baseURL string) *PagePipeline {
	return &PagePipeline{
		builder: b,
		adapter: a,
		baseURL: baseURL,
	}
}

// Execute は MangaResponse を受け取り、1枚の統合画像を生成するのだ！
func (pl *PagePipeline) Execute(ctx context.Context, manga domain.MangaResponse, characters map[string]*domain.Character) (*imgdomain.ImageResponse, error) {
	// 1. 参照URLの収集 (Runnerのロジックをここに集約)
	refURLs := pl.collectReferences(manga.Pages, characters)

	// 2. プロンプトの構築
	// 本来は builder.BuildFullPagePrompt を使うのが綺麗だけど、
	// 今は Runner にあった強力な構築ロジックをここに持ってきたのだ。
	fullPrompt := pl.buildPowerfulPrompt(manga, characters, refURLs)

	// 3. シード値の決定
	var baseSeed *int64
	if len(manga.Pages) > 0 {
		if char, ok := characters[strings.ToLower(manga.Pages[0].SpeakerID)]; ok && char.Seed > 0 {
			s := char.Seed
			baseSeed = &s
		}
	}

	// 4. リクエストの構築
	req := imgdomain.ImagePageRequest{
		Prompt:         fullPrompt,
		NegativePrompt: "deformed faces, mismatched eyes, cross-eyed, low-quality faces, merged panels, messy lineart",
		AspectRatio:    "3:4",
		Seed:           baseSeed,
		ReferenceURLs:  refURLs,
	}

	return pl.adapter.GenerateMangaPage(ctx, req)
}

// buildPowerfulPrompt は Runner にあった「日本式コマ割り」の指示を再現するのだ
func (pl *PagePipeline) buildPowerfulPrompt(manga domain.MangaResponse, characters map[string]*domain.Character, refURLs []string) string {
	var sb strings.Builder
	// ここに Runner にあった sb.WriteString("### MANDATORY FORMAT...") などの
	// 巨大なプロンプト構築ロジックを移植するのだ！
	// (中略 - RunnerのbuildUnifiedPromptの内容が入るのだ)
	return sb.String()
}

// collectReferences は必要なURLを重複なく集めるのだ
func (pl *PagePipeline) collectReferences(pages []domain.MangaPage, characters map[string]*domain.Character) []string {
	// (RunnerのcollectCharacterReferencesの内容を移植)
	return []string{}
}
