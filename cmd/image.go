package cmd

import (
	"fmt"
	"log/slog"

	"github.com/shouni/go-manga-kit/internal/config"
	"github.com/shouni/go-manga-kit/internal/pipeline"

	"github.com/spf13/cobra"
)

// imageCmd は、既存の台本JSONファイルを読み込んで画像生成フェーズを実行するためのサブコマンドなのだ。
// 台本生成をスキップして、画像生成（Phase 2）とパブリッシュ（Phase 3）のみを行うのだ。
var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "台本JSONから画像を生成して保存するのだ。",
	Long: `すでに生成・修正済みの台本JSONファイルを読み込み、漫画パネルの画像生成と保存を実行するのだ。
テキスト生成のコストを抑えつつ、画像の再生成や調整を行いたい場合に便利なのだ。`,
	RunE: imageCommand,
}

// init は、image コマンドに必要なフラグを定義し、コマンド体系に登録するための初期化関数なのだ。
func init() {
}

// imageCommand は、image サブコマンドの実行ロジック本体なのだ。
// 設定のバリデーションを行い、pipeline.ExecuteImageOnly を呼び出して一連の処理をキックするのだ。
func imageCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// --script-file がユーザーによって指定されなかった場合、
	// imageコマンド固有のデフォルト値を設定する
	if !cmd.Flags().Changed("script-file") {
		opts.ScriptFile = "examples/manga_script.json"
	}

	// 必須となる入力ファイルの存在チェック
	if opts.ScriptFile == "" {
		return fmt.Errorf("読み込むJSONファイル（--script-file）を指定してほしいのだ")
	}

	// 1. 環境変数から基本設定をロード
	cfg := config.LoadConfig()

	// 2. コマンドライン引数の値を反映
	cfg.Options = opts
	cfg.GeminiImageModel = opts.ImageModel

	slog.Info("画像生成モードを起動するのだ！",
		"input_json", cfg.Options.ScriptFile,
		"output_file", cfg.Options.OutputFile,
		"image_model", cfg.GeminiImageModel)

	// 3. パイプライン実行
	return pipeline.ExecuteImageOnly(ctx, cfg)
}
