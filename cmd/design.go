package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-manga-kit/internal/config"

	// Gemini Image Kit のドメインモデルを使用するのだ
	"github.com/shouni/go-manga-kit/internal/builder"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/pipeline"

	imgdom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/spf13/cobra"
)

var designCmd = &cobra.Command{
	Use:   "design",
	Short: "キャラクターの設定画を生成し、Seed値を確定させるのだ。",
	Long:  "Gemini Image Kit を使用して複数キャラのリファレンスを統合。一貫性のある三面図からDNA固定用のSeed値を出力するのだ。",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// characters.json を読み込み、対象キャラクターを特定
		charData, err := os.ReadFile(opts.CharacterConfig)
		if err != nil {
			return fmt.Errorf("設定ファイルの読み込みに失敗したのだ: %w", err)
		}
		// あなたのプロジェクトの domain.GetCharacters を使用
		chars, err := domain.GetCharacters(charData)
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
		imgPipe, err := pipeline.NewPipeline(httpClient, aiClient, opts.ImageModel, opts.CharacterConfig)
		if err != nil {
			return fmt.Errorf("パイプラインの初期化に失敗しました: %w", err)
		}

		var refs []string
		var descriptions []string

		// 複数キャラの情報を集約
		for _, id := range charIDs {
			char, ok := chars[id]
			if !ok {
				slog.Warn("Character not found", "charID", id)
				continue
			}
			if char.ReferenceURL != "" {
				refs = append(refs, char.ReferenceURL)
			}
			descriptions = append(descriptions, fmt.Sprintf("%s (%s)", char.Name, strings.Join(char.VisualCues, ", ")))
		}

		if len(refs) == 0 {
			return fmt.Errorf("参照できる ReferenceURL が見つからなかったのだ")
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

		// プロンプトのブラッシュアップ（ここにあなたの DefaultImagePromptSuffix の要素も混ぜたのだ！）
		designPrompt := fmt.Sprintf(
			"Masterpiece character design sheet of %s, side-by-side, multiple views (front, side, back), "+
				"standing full body, high quality, anime style, manga illustration, clean lines, vivid colors, "+
				"modern digital anime style, sharp clean lineart, vibrant flat colors, high contrast, cinematic lighting, "+
				"white background, sharp focus, 8k resolution.",
			strings.Join(descriptions, " and "),
		)

		// リクエスト作成
		pageReq := imgdom.ImagePageRequest{
			Prompt:        designPrompt,
			ReferenceURLs: refs,
			AspectRatio:   "16:9",
			Seed:          ptrInt64(seedVal),
		}

		// 統合ジェネレーターで生成
		resp, err := imgPipe.ImgGen.GenerateMangaPage(ctx, pageReq)
		if err != nil {
			slog.Error("Design generation failed", "error", err)
			return fmt.Errorf("画像の生成に失敗したのだ: %w", err)
		}

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
			extension = extensions[0] // 最も一般的な拡張子を取得 (例: ".jpeg")
		}

		// 拡張子を動的に付与してファイル名を決定
		outputName := fmt.Sprintf("design_%s%s", strings.Join(charIDs, "_"), extension)

		// 生成されたデータをローカルファイルに保存するのだ
		outputDir := "output"
		outputPath := filepath.Join(outputDir, outputName) // 保存先ディレクトリ
		if err := os.MkdirAll("output", 0755); err != nil {
			return fmt.Errorf("出力ディレクトリの作成に失敗したのだ: %w", err)
		}

		if err := os.WriteFile(outputPath, resp.Data, 0644); err != nil {
			slog.Error("Failed to save image", "path", outputPath, "error", err)
			return fmt.Errorf("画像の保存に失敗したのだ: %w", err)
		}

		// 結果表示とSeed値の出力 (フィールド名を UsedSeed に変更)
		slog.Info("Design generation completed successfully",
			slog.String("output_path", outputPath),
			slog.Int64("seed", resp.UsedSeed),
		)

		fmt.Println("\n" + strings.Repeat("✨", 25))
		fmt.Printf("🎨 デザインワーク完成: %s\n", outputPath)
		fmt.Printf("📌 抽出された Seed 値: %d\n", resp.UsedSeed) // resp.UsedSeed を使うのだ！
		fmt.Println(strings.Repeat("✨", 25))
		fmt.Printf("💡 この Seed 値を characters.json の各キャラの seed 欄に設定して、DNAを固定するのだよ！\n")

		return nil
	},
}

func init() {
	designCmd.Flags().StringSliceP("chars", "c", []string{"zundamon", "metan"}, "生成対象のキャラクターID（カンマ区切り）")
	designCmd.Flags().Int64P("seed", "s", 1000, "生成に使用するシード値。同じ値なら同じ結果が得られやすくなるのだ。")
}

func ptrInt64(v int64) *int64 { return &v }
