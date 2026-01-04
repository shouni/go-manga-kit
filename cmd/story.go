package cmd

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/shouni/go-manga-kit/internal/config"
	"github.com/shouni/go-manga-kit/internal/pipeline"

	"github.com/spf13/cobra"
)

// storyCmd は、個別パネル生成後の成果物を基に、最終的な1枚の漫画ページを錬成するのだ！
var storyCmd = &cobra.Command{
	Use:   "story",
	Short: "成果物の台本から、最終的な1枚の漫画ページを一括生成するのだ！",
	Long: `Phase 2 & 3 で出力されたMarkdown台本を読み込み、MangaPageRunner を使用して
すべてのパネルを1枚のキャンバスに収めた「完成版漫画ページ」を生成するのだ。`,
	Example: "  ap-manga-go story -f output/manga_script.md -o output/final_page.png",
	RunE:    storyCommand,
}

// init は将来的にフラグを追加する場合に使うのだ。
func init() {
}

// storyCommand は、story サブコマンドの実行ロジック本体なのだ。
func storyCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// 1. デフォルト値の判定
	// --script-file が指定されていない場合、特定のGCSパスをデフォルトにするのだ
	if !cmd.Flags().Changed("script-file") {
		opts.ScriptFile = "output/manga_plot.md"
	}

	// 必須チェック（念のためなのだ）
	if opts.ScriptFile == "" {
		return fmt.Errorf("読み込むMarkdown台本（--script-file）を指定してほしいのだ")
	}

	// 2. 設定のロード
	cfg := config.LoadConfig()

	// 3. 出力パスの安全な書き換え
	// 出力先がデフォルト（.md）のままなら、画像用（.png）に差し替えるのだ
	if opts.OutputFile == config.DefaultLocalFile || strings.HasSuffix(opts.OutputFile, ".md") {
		opts.OutputFile = "output/final_manga_page.png"
	}

	// オプションを反映
	cfg.Options = opts
	cfg.GeminiImageModel = opts.ImageModel

	slog.Info("第4ステージ（物語の集大成）を開始するのだ！",
		"input_script", cfg.Options.ScriptFile,
		"output_image", cfg.Options.OutputFile,
		"image_model", cfg.GeminiImageModel)

	// 4. パイプライン実行（物語の最終錬成なのだ！）
	if err := pipeline.ExecuteStoryOnly(ctx, cfg); err != nil {
		return fmt.Errorf("物語の最終錬成に失敗したのだ: %w", err)
	}

	slog.Info("完了なのだ！最高の1枚が完成したのだよ。これでもうボクも立派な漫画家なのだ！")
	return nil
}
