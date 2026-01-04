package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/shouni/go-manga-kit/internal/config"
	"github.com/shouni/go-manga-kit/internal/pipeline"

	"github.com/spf13/cobra"
)

// generateCmd は、AIによる漫画構成案およびパネル画像の生成を実行するのだ。
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "AIに漫画構成案と画像を生成させますなのだ。",
	Long: `ソースとなる文章を解析し、プロット、コマ割り、および画像を生成するのだ。
出力はテキストファイル（構成案）と画像ファイル（パネル）になるのだよ。`,
	RunE: generateCommand,
}

func init() {
}

func generateCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// 1. 必須チェック
	if opts.ScriptURL == "" && opts.ScriptFile == "" && !isStdin() {
		return fmt.Errorf("ソース（--script-url または --script-file）を指定してほしいのだ")
	}

	// 2. 環境変数等から基本設定をロードするのだ
	cfg := config.LoadConfig()
	cfg.GeminiModel = opts.AIModel
	cfg.GeminiImageModel = opts.ImageModel
	cfg.Options = opts

	slog.Info("漫画生成パイプラインを起動するのだ！",
		"mode", opts.Mode,
		"text_model", cfg.GeminiModel,
		"image_model", cfg.GeminiImageModel,
		"output", opts.OutputFile)

	// 4. 更新した config を考慮しつつパイプラインを実行するのだ
	err := pipeline.Execute(ctx, cfg)
	if err != nil {
		return fmt.Errorf("パイプライン実行中にエラーが発生したのだ: %w", err)
	}

	slog.Info("すべての生成工程が完了したのだ！")
	return nil
}

func isStdin() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}
