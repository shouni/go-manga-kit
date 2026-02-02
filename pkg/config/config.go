package config

import (
	"time"
)

// デフォルト値の定義なのだ
const (
	DefaultGeminiModel        = "gemini-3-flash-preview"
	DefaultImageStandardModel = "gemini-2.5-flash-image"
	DefaultImageQualityModel  = "gemini-3-pro-image-preview"
	DefaultRateInterval       = 10 * time.Second
	DefaultStyleSuffix        = "Japanese anime style, official art, cel-shaded, clean line art, high-quality manga coloring, expressive eyes, vibrant colors, cinematic lighting, masterpiece, ultra-detailed, flat shading, clear character features, no 3D effect, high resolution"
)

// Config は Go Manga Kit の各 Runner を動作させるための基本設定なのだ。
type Config struct {
	// --- AI Model Settings ---
	GeminiAPIKey       string
	GeminiModel        string
	ImageStandardModel string // 標準・高速（パネル用）
	ImageQualityModel  string // 高品質・高知能（ページ用）

	// --- Generation Settings ---
	StyleSuffix  string
	RateInterval time.Duration

	// --- Layout Settings ---
	MaxPanelsPerPage int

	// --- Timeout & Retries ---
	RequestTimeout time.Duration
}

// DefaultConfig は推奨されるデフォルト設定を返すヘルパー関数なのだ。
func DefaultConfig() Config {
	return Config{
		GeminiModel:        DefaultGeminiModel,
		ImageStandardModel: DefaultImageStandardModel,
		ImageQualityModel:  DefaultImageQualityModel,
		StyleSuffix:        DefaultStyleSuffix,
		RateInterval:       DefaultRateInterval,
	}
}
