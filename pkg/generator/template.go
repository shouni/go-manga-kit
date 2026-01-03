package generator

import (
	"fmt"
	"strings"
)

// マンガの基本構造を定義するテンプレート定数
const (
	// MangaStructureHeader はマンガとしての全体構造を強制する指示です。
	MangaStructureHeader = `### MANDATORY FORMAT: MULTI-PANEL MANGA PAGE COMPOSITION ###
- STRUCTURE: A professional Japanese manga spread with clear frame borders.
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
func BuildCharacterIdentitySection(dnas []CharacterDNA) string {
	if len(dnas) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("### CHARACTER DNA (MASTER IDENTITY) ###\n")
	for _, dna := range dnas {
		sb.WriteString(fmt.Sprintf("- [%s]: FEATURES: %s\n", dna.Name, dna.VisualCue))
	}
	sb.WriteString("\n")
	return sb.String()
}
