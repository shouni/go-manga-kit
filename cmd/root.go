package cmd

import (
	"fmt"
	"os"

	"github.com/shouni/go-manga-kit/internal/config"

	clibase "github.com/shouni/go-cli-base"
	"github.com/spf13/cobra"
)

// addAppFlags は、アプリケーション全般に適用されるグローバルフラグを定義するのだ。
// 今回は特にグローバルで持つものはないけれど、将来的に --verbose とかを入れるのに使うのだ。
func addAppFlags(rootCmd *cobra.Command) {
	// --- ソース入力関連 ---
	rootCmd.PersistentFlags().StringVarP(&opts.ScriptURL, "script-url", "u", "", "Webページからコンテンツを取得するためのURLなのだ。")
	rootCmd.PersistentFlags().StringVarP(&opts.ScriptFile, "script-file", "f", "", "入力ファイルのパス（'-'で標準入力なのだ）。")

	// --- 生成結果の出力設定 ---
	rootCmd.PersistentFlags().StringVarP(&opts.OutputFile, "output-file", "o", config.DefaultLocalFile, "保存パス（ローカル or gs://...）なのだ。")
	rootCmd.PersistentFlags().StringVarP(&opts.OutputImageDir, "output-image-dir", "i", config.DefaultLocalImageDir, "生成された画像を保存するディレクトリ（ローカル or gs://...）なのだ。")

	// --- AIモデル・挙動設定 ---
	rootCmd.PersistentFlags().StringVarP(&opts.Mode, "mode", "m", "dialogue", "プロンプト生成モードなのだ。")
	rootCmd.PersistentFlags().StringVar(&opts.AIModel, "model", config.DefaultModel, "使用する Gemini モデル名なのだ。")
	rootCmd.PersistentFlags().StringVar(&opts.ImageModel, "image-model", config.DefaultImageModel, "使用する Gemini モデル名なのだ。")
	rootCmd.PersistentFlags().DurationVar(&opts.HTTPTimeout, "http-timeout", config.DefaultHTTPTimeout, "Webリクエストのタイムアウトなのだ。")

	// --- 画像生成 (Nano Banana) 固有設定 ---
	rootCmd.PersistentFlags().IntVarP(&opts.PanelLimit, "panel-limit", "p", config.DefaultPanelLimit, "生成する漫画パネルの最大数を指定するのだ。")
	generateCmd.Flags().StringVarP(&opts.CharacterConfig, "char-config", "c", "examples/characters.json", "キャラクターの視覚情報（DNA）を定義したJSONパスなのだ。")
	//	generateCmd.Flags().StringVarP(&opts.Layout, "layout", "l", "4-panel", "漫画のレイアウト形式（4コマ、自由形式など）なのだ。")
}

// preRunAppE は、コマンド実行前に環境変数などの必須チェックを行うのだ。
func preRunAppE(cmd *cobra.Command, args []string) error {
	// Gemini APIを利用するため、APIキーの存在チェックは欠かせないのだ！
	if os.Getenv("GEMINI_API_KEY") == "" {
		return fmt.Errorf("エラー: 環境変数 GEMINI_API_KEY が設定されていません。Gemini APIの利用には必須なのだ")
	}

	return nil
}

// Execute は、アプリケーションのメインエントリポイントなのだ。
// main.go から呼び出されて、cobra のコマンドライン解析を開始するのだよ。
func Execute() {
	clibase.Execute(
		"ap-manga-go",
		addAppFlags,
		preRunAppE,
		generateCmd,
		scriptCmd,
		imageCmd,
		storyCmd,
	)
}
