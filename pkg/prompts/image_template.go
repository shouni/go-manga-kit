package prompts

import (
	"fmt"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"
)

const (
	// CinematicTags クオリティ向上のための共通タグ
	CinematicTags = "cinematic composition, high resolution, sharp focus, 4k"

	// RenderingStyle は共通の画風を定義します。
	RenderingStyle = `### GLOBAL VISUAL STYLE ###
- RENDERING: Sharp clean lineart, vibrant colors, no blurring, high contrast, cinematic manga lighting.`
)

// BuildPanelHeader は各パネルの属性（サイズや順序）を生成します。
func BuildPanelHeader(current, total int, isBigPanel bool) string {
	var sb strings.Builder
	sb.WriteString("\n===========================================\n")
	sb.WriteString(fmt.Sprintf("### [INDEPENDENT PANEL %d OF %d] ###\n", current, total))

	if isBigPanel {
		sb.WriteString("- SIZE: PRIMARY FEATURE PANEL. Large and impactful.\n")
	} else {
		sb.WriteString("- SIZE: COMPACT SUPPORTING PANEL. Integrated into the flow.\n")
	}

	sb.WriteString(fmt.Sprintf("- PLACEMENT: Part of a Right-to-Left sequence, Step %d.\n", current))
	return sb.String()
}

// BuildCharacterIdentitySection は全登場キャラの視覚的特徴をマスター定義として出力します。
func BuildCharacterIdentitySection(chars domain.CharactersMap) string {
	if len(chars) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("### CHARACTER MASTER DEFINITIONS (STRICT IDENTITY) ###\n")
	for _, char := range chars {
		cues := "None"
		if len(char.VisualCues) > 0 {
			cues = strings.Join(char.VisualCues, ", ")
		}
		sb.WriteString(fmt.Sprintf("- SUBJECT [%s]: VISUAL_FEATURES: {%s}\n", char.Name, cues))
	}
	sb.WriteString("\n")
	return sb.String()
}
