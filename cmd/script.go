package cmd

import (
	"fmt"
	"log/slog"

	"github.com/shouni/go-manga-kit/internal/config"
	"github.com/shouni/go-manga-kit/internal/pipeline"

	"github.com/spf13/cobra"
)

// scriptCmd は、台本の生成（JSON出力）のみを実行するのだ。
var scriptCmd = &cobra.Command{
	Use:   "script",
	Short: "台本（JSON）のみを生成して保存するのだ。",
	Long: `ソースとなる文章を解析し、漫画の構成案（タイトル、ページ、台詞、描写指示）を
JSON形式で出力するのだ。画像生成は行わないのだよ。`,
	RunE: scriptCommand,
}

func init() {
}

func scriptCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	// 1. 入力ソースの必須チェック (opts は addAppFlags で紐付け済みと想定)
	if opts.ScriptURL == "" && opts.ScriptFile == "" && !isStdin() {
		return fmt.Errorf("ソース（--script-url または --script-file）を指定してほしいのだ")
	}

	// --output-file がユーザーによって指定されなかった場合、
	// scriptコマンド固有のデフォルト値を設定する
	if !cmd.Flags().Changed("output-file") {
		opts.OutputFile = "output/manga_script.json"
	}

	// 2. 設定のロード
	cfg := config.LoadConfig()
	cfg.Options = opts
	cfg.GeminiModel = opts.AIModel

	slog.Info("台本生成モードを起動するのだ！",
		"mode", opts.Mode,
		"text_model", cfg.GeminiModel,
		"output", cfg.Options.OutputFile)

	// 3. 実行
	err := pipeline.ExecuteScriptOnly(ctx, cfg)
	if err != nil {
		return fmt.Errorf("台本生成中にエラーが発生したのだ: %w", err)
	}

	slog.Info("台本（JSON）の生成が完了したのだ！", "output_file", opts.OutputFile)
	return nil
}
