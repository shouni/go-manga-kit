package workflow

import (
	"time"
)

// デフォルト値の定義なのだ
const (
	DefaultGeminiModel  = "gemini-3-flash-preview"
	DefaultImageModel   = "gemini-3-pro-image-preview"
	DefaultPanelLimit   = 10
	DefaultRateInterval = 30 * time.Second
	DefaultStyleSuffix  = "Japanese anime style, official art, cel-shaded, clean line art, high-quality manga coloring, expressive eyes, vibrant colors, cinematic lighting, masterpiece, ultra-detailed, flat shading, clear character features, no 3D effect, high resolution"
)

// Config は Go Manga Kit の各 Runner を動作させるための基本設定なのだ。
type Config struct {
	// --- AI Model Settings ---
	GeminiAPIKey string
	GeminiModel  string
	ImageModel   string

	// --- Generation Settings ---
	StyleSuffix  string
	RateInterval time.Duration
	PanelLimit   int

	// --- Storage & Output Settings ---
	GCSBucket  string
	ServiceURL string

	// --- Timeout & Retries ---
	RequestTimeout time.Duration
}

// NewConfig はデフォルト値で初期化された Config を作成し、必要最小限の値をセットして返すのだ。
func NewConfig(apiKey string) Config {
	cfg := DefaultConfig()
	cfg.GeminiAPIKey = apiKey
	return cfg
}

// DefaultConfig は推奨されるデフォルト設定を返すヘルパー関数なのだ。
func DefaultConfig() Config {
	return Config{
		GeminiModel:    DefaultGeminiModel,
		ImageModel:     DefaultImageModel,
		StyleSuffix:    DefaultStyleSuffix,
		RateInterval:   DefaultRateInterval,
		PanelLimit:     DefaultPanelLimit,
		RequestTimeout: 5 * time.Minute,
	}
}
