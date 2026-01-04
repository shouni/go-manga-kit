package runner

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"github.com/shouni/go-manga-kit/internal/config"
	mngdom "github.com/shouni/go-manga-kit/pkg/domain"

	imagekit "github.com/shouni/gemini-image-kit/pkg/adapters"
	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// ImageRunner は、漫画の台本データを基に画像を生成するためのインターフェース。
type ImageRunner interface {
	// Run は台本の全ページに対して画像生成を実行し、結果のリストを返す。
	Run(ctx context.Context, manga mngdom.MangaResponse) ([]*imagedom.ImageResponse, error)
}

// MangaImageRunner は、キャラクターの一貫性を保ちながら並列で画像生成を行う実体。
type MangaImageRunner struct {
	imageAdapter imagekit.ImageAdapter       // 画像生成AI（Imagen/Gemini）へのアダプター
	characters   map[string]mngdom.Character // 利用可能なキャラクター設定のマップ
	limit        int                         // 生成する最大パネル数の制限
	basePrompt   string                      // 全パネル共通で適用する画風（スタイル）の指示
}

// NewMangaImageRunner は、MangaImageRunnerの新しいインスタンスを生成して返す。
func NewMangaImageRunner(adapter imagekit.ImageAdapter, chars map[string]mngdom.Character, limit int, basePrompt string) *MangaImageRunner {
	normalizedChars := make(map[string]mngdom.Character)
	for k, v := range chars {
		normalizedChars[strings.ToLower(k)] = v
	}

	return &MangaImageRunner{
		imageAdapter: adapter,
		characters:   normalizedChars,
		limit:        limit,
		basePrompt:   basePrompt,
	}
}

// Run は並列処理を用いて、各ページの画像を生成するメインロジックなのだ。
func (ir *MangaImageRunner) Run(ctx context.Context, manga mngdom.MangaResponse) ([]*imagedom.ImageResponse, error) {
	pages := manga.Pages
	// 指定があれば、生成するパネル数を制限するのだ（テスト用などに便利！）
	if ir.limit > 0 && len(pages) > ir.limit {
		slog.Info("パネル数に制限を適用したのだ", "limit", ir.limit, "total", len(pages))
		pages = pages[:ir.limit]
	}

	images := make([]*imagedom.ImageResponse, len(pages))
	eg, egCtx := errgroup.WithContext(ctx)

	// 設定ファイルから取得した間隔で、レートリミット（流量制限）をかけるのだ
	// Burst 2 により、開始直後に2枚までは同時にリクエストを開始できるのだ
	limiter := rate.NewLimiter(rate.Every(config.DefaultRateLimit), 2)
	slog.Info("並列画像生成を開始するのだ", "count", len(pages), "interval", config.DefaultRateLimit)

	for i, page := range pages {
		i, page := i, page // ゴルーチンのクロージャ対策なのだ

		eg.Go(func() error {
			// 1. レートリミットに従って、自分の番が来るまで待機するのだ
			if err := limiter.Wait(egCtx); err != nil {
				return err
			}

			// 2. ページ情報から、描画すべきキャラクターを特定するのだ
			resolvedID := ir.resolveID(page)
			char := ir.getChar(resolvedID)

			// 3. 画風、キャラクターのDNA、シーンの指示を組み合わせてプロンプトを作るのだ
			prompt, negPrompt := ir.buildPrompt(page.VisualAnchor, char.VisualCues)

			slog.Info("パネルを生成中...", "page", i+1, "character", char.Name, "resolved_id", resolvedID)

			// 4. アダプターを介してAIに画像生成を依頼するのだ
			// キャラクターのシード値を int32 にキャストしてポインタにするのだ
			var seedPtr *int64
			if char.Seed > 0 {
				if char.Seed > math.MaxInt32 {
					slog.Warn("シード値がint32の最大値を超えているため、切り捨てられます", "original_seed", char.Seed, "max_value", math.MaxInt32)
				}
				seedPtr = &char.Seed
			}

			resp, err := ir.imageAdapter.GenerateMangaPanel(egCtx, imagedom.ImageGenerationRequest{
				Prompt:         prompt,
				NegativePrompt: negPrompt,
				Seed:           seedPtr,
				ReferenceURL:   char.ReferenceURL,
				AspectRatio:    "16:9",
			})
			if err != nil {
				slog.Error("パネル生成に失敗したのだ", "page", i+1, "error", err)
				return err
			}

			images[i] = resp
			slog.Info("パネル生成に成功したのだ", "page", i+1)
			return nil
		})
	}

	// すべての並列処理が完了するのを待つのだ
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	slog.Info("すべてのパネルが正常に生成されたのだ", "total", len(images))
	return images, nil
}

// resolveID は、MangaPageの情報（SpeakerIDや本文）から最適なキャラクターIDを決定するのだ。
func (ir *MangaImageRunner) resolveID(page mngdom.MangaPage) string {
	// 1. SpeakerID が指定されていれば、それを最優先で使うのだ
	if page.SpeakerID != "" {
		return strings.ToLower(page.SpeakerID)
	}

	// 2. なければ VisualAnchor（描写指示）の中のキーワードから推測を試みるのだ
	anchor := strings.ToLower(page.VisualAnchor)
	if strings.Contains(anchor, "metan") {
		return "metan"
	}
	if strings.Contains(anchor, "zundamon") {
		return "zundamon"
	}

	return ""
}

// getChar は、IDに一致するキャラクター設定を取得するのだ。見つからない場合はデフォルトを返すのだ。
func (ir *MangaImageRunner) getChar(id string) mngdom.Character {
	if c, ok := ir.characters[id]; ok {
		return c
	}
	// フォールバック: ずんだもんを優先して探すのだ
	if zunda, ok := ir.characters["zundamon"]; ok {
		return zunda
	}
	// 最終手段として、登録されている誰か一人を返すのだ
	for _, v := range ir.characters {
		return v
	}
	return mngdom.Character{Name: "Unknown"}
}

// buildPrompt は、ポジティブ（描きたいもの）とネガティブ（描きたくないもの）の指示を構築するのだ。
func (ir *MangaImageRunner) buildPrompt(anchor string, cues []string) (string, string) {
	// ポジティブプロンプト: スタイル、キャラクターの特徴、シーンの状況を合成するのだ
	positive := fmt.Sprintf("%s, %s, %s, cinematic composition, high resolution, no speech bubbles",
		ir.basePrompt,
		strings.Join(cues, ", "),
		anchor,
	)

	// ネガティブプロンプト: 吹き出し、文字、低品質な描写を徹底的に排除するのだ
	negative := "speech bubble, dialogue balloon, text, alphabet, letters, words, signatures, watermark, username, low quality, distorted, bad anatomy"

	return positive, negative
}
