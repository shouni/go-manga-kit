package config

import (
	"time"

	"github.com/shouni/go-utils/envutil"
)

// デフォルト値の定義なのだ
const (
	DefaultModel             = "gemini-3-flash-preview"
	DefaultImageModel        = "gemini-3-pro-image-preview"
	DefaultHTTPTimeout       = 30 * time.Second
	DefaultPanelLimit        = 10
	DefaultRateLimit         = 30 * time.Second
	DefaultCharactersFile    = "internal/config/characters.json" // キャラクターの視覚情報（DNA）を定義したJSONパス
	DefaultLocalFile         = "output/manga_plot.md"            // パブリッシャーで使用するデフォルト保存先なのだ
	DefaultLocalImageDir     = "output/images"                   // パブリッシャーで使用するデフォルト保存先なのだ
	DefaultImagePromptSuffix = "Japanese anime style, official art, cel-shaded, clean line art, high-quality manga coloring, expressive eyes, vibrant colors, cinematic lighting, masterpiece, ultra-detailed, flat shading, clear character features, no 3D effect, high resolution"
)

// Config はアプリケーション全体の環境設定（APIキーやクラウド設定）を保持する構造体なのだ。
type Config struct {
	ProjectID         string
	LocationID        string
	GeminiAPIKey      string
	GeminiModel       string
	GeminiImageModel  string
	ImagePromptSuffix string

	Options GenerateOptions
}

// LoadConfig は環境変数から設定を読み込み、構造体を返すのだ！
func LoadConfig() *Config {
	cfg := &Config{
		ProjectID:         envutil.GetEnv("PROJECT_ID", ""),
		LocationID:        envutil.GetEnv("REGION", ""),
		GeminiAPIKey:      envutil.GetEnv("GEMINI_API_KEY", ""),
		GeminiModel:       envutil.GetEnv("GEMINI_MODEL", DefaultModel),
		GeminiImageModel:  envutil.GetEnv("IMAGE_GEMINI_MODEL", DefaultImageModel),
		ImagePromptSuffix: envutil.GetEnv("IMAGE_PROMPT_SUFFIX", DefaultImagePromptSuffix),
	}
	return cfg
}

// GenerateOptions は CLI フラグから渡される実行時のパラメータなのだ。
type GenerateOptions struct {
	// ソース入力関連
	ScriptURL       string // --script-url
	ScriptFile      string // --script-file
	OutputFile      string // --output-file
	PanelLimit      int
	CharacterConfig string
	Layout          string

	// 画像生成関連
	OutputImageDir string // --output-image-dir

	// AI挙動設定
	AIModel    string // --model: テキスト生成用のGeminiモデル
	ImageModel string // --image-model: 画像生成用のGeminiモデル
	Mode       string // --mode

	// 実行制御
	HTTPTimeout time.Duration // --http-timeout
}
