package prompts

import "strings"

const (
	// CinematicTags クオリティ向上のための共通タグ
	CinematicTags = "cinematic composition, high resolution, sharp focus, 4k"

	// RenderingStyle は共通の画風を定義します。
	RenderingStyle = `### GLOBAL VISUAL STYLE ###
- RENDERING: Sharp clean lineart, vibrant colors, no blurring, high contrast, cinematic manga lighting.`
)

// sanitizeInline は文字列をプロンプトに埋め込む前の最低限の正規化を行います。
func sanitizeInline(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.TrimSpace(s)
}

// formatDialogue はダイアログの正規化のみを行います。
func formatDialogue(s string) string {
	s = sanitizeInline(s)
	// AIの混乱を防ぐため、ダブルクォートをシングルクォートに逃がします
	s = strings.ReplaceAll(s, "\"", "'")
	return s
}
