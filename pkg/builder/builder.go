package builder

import (
	"context"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	imageKit "github.com/shouni/gemini-image-kit/pkg/adapters"
	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-manga-kit/pkg/adapters"
	"google.golang.org/genai"
)

// InitializeAIClient は gemini クライアントを初期化します。
func InitializeAIClient(ctx context.Context, apiKey string) (gemini.GenerativeModel, error) {
	const defaultGeminiTemperature = float32(0.2)
	clientConfig := gemini.Config{
		APIKey:      apiKey,
		Temperature: genai.Ptr(defaultGeminiTemperature),
	}
	aiClient, err := gemini.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("AIクライアントの初期化に失敗しました: %w", err)
	}
	return aiClient, nil
}

// InitializeImageCore は各アダプターで共有する画像処理コアを生成します。
func InitializeImageCore(clientInterface httpkit.ClientInterface) imageKit.ImageGeneratorCore {
	// 参照画像のダウンロード結果を保持するキャッシュ
	imgCache := cache.New(30*time.Minute, 1*time.Hour)
	cacheTTL := 1 * time.Hour

	return imageKit.NewGeminiImageCore(
		clientInterface,
		imgCache,
		cacheTTL,
	)
}

// InitializeImageAdapter は、個別パネル用アダプターの初期化。
func InitializeImageAdapter(core imageKit.ImageGeneratorCore, aiClient gemini.GenerativeModel, imageModel, promptSuffix string) (adapters.ImageAdapter, error) {
	imageAdapter, err := imageKit.NewGeminiImageAdapter(
		core,
		aiClient,
		imageModel,
		promptSuffix,
	)
	if err != nil {
		return nil, fmt.Errorf("画像アダプターの初期化に失敗しました: %w", err)
	}

	return imageAdapter, nil
}

// InitializeMangaPageAdapter は、マンガのページ画像を生成するアダプターの初期化。
// adapters.MangaPageAdapter インターフェースに適合させて返すのだ。
func InitializeMangaPageAdapter(core imageKit.ImageGeneratorCore, aiClient gemini.GenerativeModel, imageModel string) adapters.MangaPageAdapter {
	// pipeline.MangaPageAdapter インターフェースを実装しているアダプターを返すのだ
	return imageKit.NewGeminiMangaPageAdapter(
		core,
		aiClient,
		imageModel,
	)
}

//// NewPromptBuilder は、キャラクター情報と画風サフィックスを元にプロンプトビルダーを生成するのだ！
//// 戻り値は pipeline パッケージで定義されたインターフェース（pipeline.PromptBuilder）として返すのだ。
//func NewPromptBuilder(chars []domain.Character, promptSuffix string) pipeline.PromptBuilder {
//	// 1. スライス形式のキャラクター情報を、IDや名前で引きやすい CharactersMap に変換するのだ。
//	// これにより、generator パッケージ側で O(1) でキャラクターを検索できるようになるのだ！
//	charMap := domain.BuildCharactersMap(chars)
//
//	// 2. generator パッケージのコンストラクタを呼び出すのだ。
//	// 先ほど記憶した func NewPromptBuilder(chars domain.CharactersMap, suffix string) *PromptBuilder に完全準拠なのだ。
//	return generator.NewPromptBuilder(charMap, promptSuffix)
//}
