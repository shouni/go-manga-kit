package prompt

import (
	"fmt"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"
)

// マンガの基本構造を定義するテンプレート定数
const (
	// MangaStructureHeader はマンガとしての全体構造を強制する指示です。
	MangaStructureHeader = `### MANDATORY FORMAT: MULTI-PANEL MANGA PAGE COMPOSITION ###
- STRUCTURE: A p≈with clear frame borders.
- READING ORDER: Right-to-Left, Top-to-Bottom.
- GUTTERS: Ultra-thin, crisp hairline dividers. NO OVERLAPPING. Each panel is a separate scene.`

	// RenderingStyle は描画の品質を一貫させるための指示です。
	RenderingStyle = `### GLOBAL VISUAL STYLE ###
- RENDERING: Sharp clean lineart, vibrant colors, no blurring, high contrast, cinematic manga lighting.`
)

// BuildPanelHeader は個別のコマに対するヘッダー指示を生成します。
func BuildPanelHeader(current, total int, isBigPanel bool) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n===========================================\n"))
	sb.WriteString(fmt.Sprintf("### [INDEPENDENT PANEL %d OF %d] ###\n", current, total))

	if isBigPanel {
		sb.WriteString("- SIZE: PRIMARY FEATURE PANEL. Large and impactful.\n")
	} else {
		sb.WriteString("- SIZE: COMPACT SUPPORTING PANEL. Integrated into the flow.\n")
	}

	sb.WriteString(fmt.Sprintf("- PLACEMENT: Part of a Right-to-Left sequence, Step %d.\n", current))
	return sb.String()
}

// BuildCharacterIdentitySection はキャラクターのマスター情報を定義するセクションを生成します。
func BuildCharacterIdentitySection(chars []domain.Character) string {
	if len(chars) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("### CHARACTER MASTER DEFINITIONS (STRICT IDENTITY) ###\n")
	for _, char := range chars {
		// VisualCues (スライス) をカンマ区切りで結合する
		cues := "None"
		if len(char.VisualCues) > 0 {
			cues = strings.Join(char.VisualCues, ", ")
		}

		// AIが「この名前が出てきたらこの特徴を使う」と認識しやすい形式にするのだ
		sb.WriteString(fmt.Sprintf("- SUBJECT [%s]: VISUAL_FEATURES: {%s}\n", char.Name, cues))
	}
	sb.WriteString("\n")
	return sb.String()
}
