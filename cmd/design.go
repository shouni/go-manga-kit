package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"mime"
	"os"
	"path/filepath"
	"strings"

	imgdom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-manga-kit/internal/builder"
	"github.com/shouni/go-manga-kit/internal/config"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/spf13/cobra"
)

var designCmd = &cobra.Command{
	Use:   "design",
	Short: "キャラクターの設定画を生成し、Seed値を確定させるのだ。",
	Long:  "Gemini Image Kit を使用して複数キャラのリファレンスを統合。一貫性のある三面図からDNA固定用のSeed値を出力するのだ。",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// characters.json を読み込み、対象キャラクターを特定
		chars, err := loadCharacters(opts.CharacterConfig)
		if err != nil {
			return err
		}

		charIDs, err := cmd.Flags().GetStringSlice("chars")
		if err != nil {
			return fmt.Errorf("--chars フラグの解析に失敗しました: %w", err)
		}
		if len(charIDs) == 0 {
			return fmt.Errorf("--chars で最低1人のキャラクターIDを指定してほしいのだ")
		}

		aiClient, err := builder.InitializeAIClient(ctx, os.Getenv("GEMINI_API_KEY"))
		if err != nil {
			return fmt.Errorf("AIクライアントの初期化に失敗しました: %w", err)
		}
		httpClient := httpkit.New(config.DefaultHTTPTimeout)
		mangaGen, err := generator.NewMangaGenerator(httpClient, aiClient, opts.ImageModel, opts.CharacterConfig)
		if err != nil {
			return fmt.Errorf("MangaGeneratorの初期化に失敗しました: %w", err)
		}

		// 複数キャラの情報を集約
		refs, descriptions, err := collectCharacterAssets(chars, charIDs)
		if err != nil {
			return err
		}
		slog.Info("Executing design work generation",
			slog.Any("chars", charIDs),
			slog.Int("ref_count", len(refs)),
		)

		// フラグからシード値を取得するのだ
		seedVal, err := cmd.Flags().GetInt64("seed")
		if err != nil {
			return fmt.Errorf("seedフラグの解析に失敗したのだ: %w", err)
		}

		// プロンプトを生成
		designPrompt := buildDesignPrompt(descriptions)
		// リクエスト作成
		pageReq := imgdom.ImagePageRequest{
			Prompt:        designPrompt,
			ReferenceURLs: refs,
			AspectRatio:   "16:9",
			Seed:          ptrInt64(seedVal),
		}

		// 統合ジェネレーターで生成
		resp, err := mangaGen.ImgGen.GenerateMangaPage(ctx, pageReq)
		if err != nil {
			slog.Error("Design generation failed", "error", err)
			return fmt.Errorf("画像の生成に失敗したのだ: %w", err)
		}

		outputPath, err := saveResponseImage(*resp, charIDs, "output")
		if err != nil {
			slog.Error("Failed to save image", "error", err)
			return fmt.Errorf("画像の保存に失敗しました: %w", err)
		}
		// 結果表示とSeed値の出力 (フィールド名を UsedSeed に変更)
		slog.Info("Design generation completed successfully",
			slog.String("output_path", outputPath),
			slog.Int64("seed", resp.UsedSeed),
		)
		printSuccessMessage(outputPath, resp.UsedSeed)

		return nil
	},
}

func init() {
	designCmd.Flags().StringSliceP("chars", "c", []string{"zundamon", "metan"}, "生成対象のキャラクターID（カンマ区切り）")
	designCmd.Flags().Int64P("seed", "s", 1000, "生成に使用するシード値。同じ値なら同じ結果が得られやすくなるのだ。")
}

func loadCharacters(path string) (map[string]domain.Character, error) {
	charData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("設定ファイルの読み込み失敗なのだ: %w", err)
	}
	return domain.GetCharacters(charData)
}

// プロンプトのブラッシュアップ
func buildDesignPrompt(descriptions []string) string {
	// TODO:旧型あとで確認して削除
	//designPrompt := fmt.Sprintf(
	//	"Masterpiece character design sheet of %s, side-by-side, multiple views (front, side, back), "+
	//		"standing full body, high quality, anime style, manga illustration, clean lines, vivid colors, "+
	//		"modern digital anime style, sharp clean lineart, vibrant flat colors, high contrast, cinematic lighting, "+
	//		"white background, sharp focus, 4k resolution, highly detailed.",
	//	strings.Join(descriptions, " and "),
	//)
	base := fmt.Sprintf("Masterpiece character design sheet of %s, side-by-side, multiple views (front, side, back), standing full body",
		strings.Join(descriptions, " and "))

	// config等からスタイルを取得できるようになるとさらに良い
	return fmt.Sprintf("%s, %s, white background, sharp focus, 4k resolution", base, config.DefaultImagePromptSuffix)
}

// collectCharacterAssets collects the reference URLs and descriptions for the specified characters.
func collectCharacterAssets(chars map[string]domain.Character, ids []string) ([]string, []string, error) {
	var refs []string
	var descs []string

	for _, id := range ids {
		char, ok := chars[id]
		if !ok {
			slog.Warn("Character not found", "charID", id)
			continue
		}
		if char.ReferenceURL != "" {
			refs = append(refs, char.ReferenceURL)
		}
		descs = append(descs, fmt.Sprintf("%s (%s)", char.Name, strings.Join(char.VisualCues, ", ")))
	}

	if len(refs) == 0 {
		return nil, nil, fmt.Errorf("参照URLを持つキャラが一人もいないのだ")
	}
	return refs, descs, nil
}

// ptrInt64 returns a pointer to the given int64 value.
func ptrInt64(v int64) *int64 { return &v }

// saveResponseImage saves the image response to a file in the specified directory.
func saveResponseImage(resp imgdom.ImageResponse, charIDs []string, dir string) (string, error) {
	// MIMEタイプから拡張子を決定
	var extension string
	extensions, err := mime.ExtensionsByType(resp.MimeType)
	if err != nil || len(extensions) == 0 {
		slog.Warn(
			"Could not determine file extension from MIME type, defaulting to .png",
			slog.String("mime_type", resp.MimeType),
		)
		extension = ".png" // フォールバック
	} else {
		// なるべく一般的なものを選ぶ
		extension = extensions[0]
		for _, ext := range extensions {
			if ext == ".png" || ext == ".jpeg" || ext == ".jpg" {
				extension = ext
				break
			}
		}
	}

	filename := fmt.Sprintf("design_%s%s", strings.Join(charIDs, "_"), extension)
	path := filepath.Join(dir, filename)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, resp.Data, 0644); err != nil {
		return "", err
	}
	return path, nil
}

// printSuccessMessage outputs a formatted success message after design generation, including the output path and seed value.
func printSuccessMessage(path string, seed int64) {
	fmt.Println("\n" + strings.Repeat("✨", 25))
	fmt.Printf("🎨 デザインワーク完成: %s\n", path)
	fmt.Printf("📌 抽出された Seed 値: %d\n", seed)
	fmt.Println(strings.Repeat("✨", 25))
	fmt.Println("💡 この Seed 値を characters.json に設定してDNAを固定するのだ！")
}
