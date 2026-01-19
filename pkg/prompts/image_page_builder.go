package prompts

import (
	"fmt"
	"sort"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"
)

type ResourceMap struct {
	CharacterFiles map[string]int
	PanelFiles     map[string]int
	OrderedURIs    []string
	OrderedURLs    []string
}

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

const (
	// NegativePagePrompt は生成から除外したい要素を定義します。
	NegativePagePrompt = "monochrome, black and white, greyscale, screentone, hatching, dot shades, ink sketch, line art only, realistic photos, 3d render, watermark, signature, deformed faces, bad anatomy, disfigured, poorly drawn hands, extra panels, unexpected panels, more than specified panels, split panels"

	// MangaStructureHeader は漫画の構造に関する基本ルールを定義します。
	MangaStructureHeader = `### FORMAT RULES: FULL COLOR ANIME MANGA ###
- STYLE: Vibrant Full Color Digital Anime Style. High saturation, cinematic lighting.
- RENDERING: Sharp clean lineart with professional digital coloring. NO screentones.
- LAYOUT: Strict multi-panel composition. Use ONLY the specified number of panels.
- NO FILLER: Do not add extra panels or decorative small frames. Fill the page with the given count.
- BORDERS: Deep black, crisp frame borders for EVERY panel.
- GUTTERS: Pure white space between panels.
- READING FLOW: Right-to-Left, Top-to-Bottom.`
)

// BuildMangaPagePrompt は漫画の1ページを生成するためのシステムプロンプトとユーザープロンプトを構築します。
func (pb *ImagePromptBuilder) BuildMangaPagePrompt(panels []domain.Panel, rm *ResourceMap) (userPrompt string, systemPrompt string) {
	const mangaSystemInstruction = "You are a master digital artist. You MUST follow the exact panel count and layout rules. Character identity MUST match the character master reference files."

	systemParts := []string{
		mangaSystemInstruction,
		MangaStructureHeader,
		RenderingStyle,
		CinematicTags,
	}
	if pb.defaultSuffix != "" {
		systemParts = append(systemParts, fmt.Sprintf("### ARTISTIC STYLE ###\n%s", pb.defaultSuffix))
	}
	systemPrompt = strings.Join(systemParts, "\n\n")

	var us strings.Builder
	numPanels := len(panels)

	// --- 1. 基本要求 ---
	us.WriteString("# FULL COLOR PAGE PRODUCTION REQUEST\n")
	us.WriteString("- OUTPUT: ONE single portrait manga page image.\n")
	us.WriteString("- COLOR: STRICTLY VIBRANT FULL COLOR. NO monochrome, NO screentones.\n")
	us.WriteString(fmt.Sprintf("- PANEL COUNT: [ %d ] (STRICTLY ONLY %d PANELS. DO NOT ADD ANY MORE).\n\n", numPanels, numPanels))

	// --- 2. レイアウト指示 ---
	us.WriteString("## MANDATORY PAGE STRUCTURE\n")
	us.WriteString("- OUTPUT FORMAT: A single vertical manga page.\n")
	us.WriteString("- GRID SYSTEM: 2-column grid (except for single panel pages).\n")
	us.WriteString("- READING ORDER: Japanese Style (Right-to-Left, then Top-to-Bottom).\n")

	us.WriteString("- PANEL PLACEMENT MAP:\n")
	if numPanels == 1 {
		us.WriteString("  * PANEL 1: SINGLE FULL-PAGE PANEL (covers entire image area).\n")
	} else {
		for i := 0; i < numPanels; i++ {
			panelNum := i + 1
			if numPanels%2 == 1 && i == numPanels-1 {
				us.WriteString(fmt.Sprintf("  * PANEL %d: BOTTOM ROW, FULL-WIDTH (spans across both columns).\n", panelNum))
			} else {
				row := (i / 2) + 1
				side := "RIGHT"
				if i%2 == 1 {
					side = "LEFT"
				}
				us.WriteString(fmt.Sprintf("  * PANEL %d: ROW %d, %s column.\n", panelNum, row, side))
			}
		}
	}

	if numPanels == 1 {
		us.WriteString("- PANEL SIZE: One large dramatic cinematic frame.\n")
	} else if numPanels > 1 && numPanels%2 == 1 {
		us.WriteString(fmt.Sprintf("- SPECIAL RULE: PANEL %d is a wide cinematic panel at the bottom. All other panels are standard sized in 2 columns.\n", numPanels))
	} else {
		us.WriteString("- PANEL SIZE: All panels are standard, uniform, and balanced in the 2-column grid.\n")
	}

	us.WriteString("- FRAME STYLE: Deep black, crisp borders for EVERY panel. NO overlapping frames.\n")
	us.WriteString("- GUTTERS: Pure white gutters between all panels.\n")
	us.WriteString("- ABSOLUTE BAN: Do NOT add extra frames, decorative small panels, or any panels inside other panels.\n\n")

	// --- 3. キャラクター参照設定 (順序の安定化) ---
	us.WriteString("## CHARACTER MASTER REFERENCES (FIXED IDENTITY + COLOR PALETTE)\n")

	type charRef struct {
		id  string
		idx int
	}
	refs := make([]charRef, 0, len(rm.CharacterFiles))
	for id, idx := range rm.CharacterFiles {
		refs = append(refs, charRef{id: id, idx: idx})
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].idx < refs[j].idx })

	for _, r := range refs {
		sID := r.id
		fileIdx := r.idx
		displayName := sID
		visualDesc := "vivid anime color palette"

		if char := pb.characterMap.GetCharacter(sID); char != nil {
			displayName = char.Name
			if len(char.VisualCues) > 0 {
				visualDesc = strings.Join(char.VisualCues, ", ")
			}
		}
		us.WriteString(fmt.Sprintf(
			"- SUBJECT [%s]: Identity (face, hair, eyes, colors) MUST match input_file_%d exactly. Traits: {%s}. This reference OVERRIDES seed/style drift.\n",
			displayName, fileIdx, visualDesc,
		))
	}
	us.WriteString("\n")

	// --- 4. パネルごとの詳細指示 ---
	// bigPanelIndex の整合性を確保
	bigPanelIndex := -1
	if numPanels == 1 {
		bigPanelIndex = 0
	} else if numPanels > 1 && numPanels%2 == 1 {
		bigPanelIndex = numPanels - 1
	}

	us.WriteString("## PANEL BREAKDOWN (STRICT COUNT + FIXED LAYOUT)\n")
	for i, panel := range panels {
		panelNum := i + 1

		panelSizeLabel := "Standard"
		if i == bigPanelIndex {
			if numPanels == 1 {
				panelSizeLabel = "FULL-PAGE"
			} else {
				panelSizeLabel = "FULL-WIDTH IMPACT (spans both columns)"
			}
		}

		status := ""
		if i == numPanels-1 && numPanels > 1 {
			status = " - FINAL PANEL"
		}
		us.WriteString(fmt.Sprintf("### PANEL %d [%s]%s\n", panelNum, panelSizeLabel, status))

		// 位置指示の不整合を排除
		if numPanels == 1 {
			us.WriteString("- POSITION: Entire page area.\n")
		} else if i == bigPanelIndex {
			us.WriteString("- POSITION: Bottom row, full width.\n")
		} else {
			row := (i / 2) + 1
			side := "RIGHT"
			if i%2 == 1 {
				side = "LEFT"
			}
			us.WriteString(fmt.Sprintf("- POSITION: Row %d, %s column.\n", row, side))
		}

		// キャラクター名の決定とファイル参照
		displayName := panel.SpeakerID
		charFileIdx := -1
		if char := pb.characterMap.GetCharacter(panel.SpeakerID); char != nil {
			displayName = char.Name
			if idx, ok := rm.CharacterFiles[char.ID]; ok {
				charFileIdx = idx
			}
		}

		// シーン記述のサニタイズと置換
		sceneDescription := sanitizeInline(panel.VisualAnchor)
		sceneDescription = strings.ReplaceAll(sceneDescription, panel.SpeakerID, displayName)

		charCues := ""
		if charFileIdx != -1 {
			if char := pb.characterMap.GetCharacter(panel.SpeakerID); char != nil && len(char.VisualCues) > 0 {
				charCues = fmt.Sprintf(" (Identity MUST match input_file_%d: %s)", charFileIdx, strings.Join(char.VisualCues, ", "))
			} else {
				charCues = fmt.Sprintf(" (Identity MUST match input_file_%d)", charFileIdx)
			}
		}

		us.WriteString("- RENDER: FULL COLOR.\n")
		us.WriteString(fmt.Sprintf("- SUBJECT: %s\n", displayName))
		us.WriteString(fmt.Sprintf("- ACTION: %s%s\n", sceneDescription, charCues))

		// ポーズ参照 (ReferenceURL)
		if panel.ReferenceURL != "" {
			if fileIdx, ok := rm.PanelFiles[panel.ReferenceURL]; ok {
				us.WriteString(fmt.Sprintf("- POSE_REF: Use input_file_%d for BODY/POSE/ANATOMY only. IGNORE face/hair/colors from this file.\n", fileIdx))
				if charFileIdx != -1 {
					us.WriteString(fmt.Sprintf("- IDENTITY_FIX: Face/hair/eyes MUST match input_file_%d exactly.\n", charFileIdx))
				}
			}
		}

		// セリフ指示
		if panel.Dialogue != "" {
			cleanText := formatDialogue(panel.Dialogue)
			us.WriteString(fmt.Sprintf("- SPEECH: Speech bubble for [%s].\n", displayName))
			us.WriteString(fmt.Sprintf("  - TEXT_TO_RENDER: \"%s\"\n", cleanText))
			us.WriteString("  - TYPOGRAPHY: Use professional Japanese manga font.\n")
			us.WriteString("  - LANGUAGE: Japanese characters. Ensure accuracy and legibility.\n")
			us.WriteString("  - STYLE: High-quality typesetting.\n")
		}

		if i == numPanels-1 {
			us.WriteString("- STOP: End of page. Do not draw any additional panels/frames after this.\n")
		}
		us.WriteString("\n")
	}

	userPrompt = us.String()
	return userPrompt, systemPrompt
}
