package config

import (
	"time"
)

// デフォルト値の定義なのだ
const (
	DefaultGeminiModel    = "gemini-3-flash-preview"
	DefaultImageModel     = "gemini-3-pro-image-preview"
	DefaultRateInterval   = 30 * time.Second
	DefaultStyleSuffix    = "Japanese anime style, official art, cel-shaded, clean line art, high-quality manga coloring, expressive eyes, vibrant colors, cinematic lighting, masterpiece, ultra-detailed, flat shading, clear character features, no 3D effect, high resolution"
	DefaultRequestTimeout = 5 * time.Minute
	// DefaultPageFileName は共通のベースファイル名
	DefaultPageFileName = "manga_page.png"
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

	// --- Timeout & Retries ---
	RequestTimeout time.Duration
}

// DefaultConfig は推奨されるデフォルト設定を返すヘルパー関数なのだ。
func DefaultConfig() Config {
	return Config{
		GeminiModel:    DefaultGeminiModel,
		ImageModel:     DefaultImageModel,
		StyleSuffix:    DefaultStyleSuffix,
		RateInterval:   DefaultRateInterval,
		RequestTimeout: DefaultRequestTimeout,
	}
}
