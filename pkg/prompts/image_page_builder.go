package prompts

import (
	"fmt"
	"math/rand/v2"
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

// 文字列をプロンプトに埋め込む前の最低限の正規化（改行・ダブルクォート事故を減らす）
func sanitizeInline(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.TrimSpace(s)
	return s
}

// ダイアログの正規化のみを行う（記号は付けない）
func formatDialogue(s string) string {
	s = sanitizeInline(s)
	// AIが混乱するため、中のダブルクォートはシングルクォートに逃がす
	s = strings.ReplaceAll(s, "\"", "'")
	return s
}

const (
	// NegativePagePrompt
	NegativePagePrompt = "monochrome, black and white, greyscale, screentone, hatching, dot shades, ink sketch, line art only, realistic photos, 3d render, watermark, signature, deformed faces, bad anatomy, disfigured, poorly drawn hands, extra panels, unexpected panels, more than specified panels, split panels"

	// MangaStructureHeader
	MangaStructureHeader = `### FORMAT RULES: FULL COLOR ANIME MANGA ###
- STYLE: Vibrant Full Color Digital Anime Style. High saturation, cinematic lighting.
- RENDERING: Sharp clean lineart with professional digital coloring. NO screentones.
- LAYOUT: Strict multi-panel composition. Use ONLY the specified number of panels.
- NO FILLER: Do not add extra panels or decorative small frames. Fill the page with the given count.
- BORDERS: Deep black, crisp frame borders for EVERY panel.
- GUTTERS: Pure white space between panels.
- READING FLOW: Right-to-Left, Top-to-Bottom.`
)

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
	us.WriteString("# FULL COLOR PAGE PRODUCTION REQUEST\n")
	us.WriteString("- OUTPUT: ONE single portrait manga page image.\n")
	us.WriteString("- COLOR: STRICTLY VIBRANT FULL COLOR. NO monochrome, NO screentones.\n")
	us.WriteString(fmt.Sprintf("- PANEL COUNT: [ %d ] (STRICTLY ONLY %d PANELS. DO NOT ADD ANY MORE).\n", len(panels), len(panels)))

	// レイアウト固定（2列グリッド）
	us.WriteString("- LAYOUT GRID: 2 columns. Reading order is Right-to-Left within each row, Top-to-Bottom across rows.\n")
	if len(panels) > 1 && len(panels)%2 == 1 {
		us.WriteString(fmt.Sprintf("- LAST PANEL RULE: PANEL %d is FULL-WIDTH at the bottom (spans both columns). All other panels are standard size.\n", len(panels)))
	} else {
		us.WriteString("- PANEL SIZE RULE: All panels are standard and uniform. No inset panels, no split panels.\n")
	}
	us.WriteString("- BORDERS: deep black crisp borders for EVERY panel. GUTTERS: pure white.\n")
	us.WriteString("- ABSOLUTE BAN: no extra frames, no decorative mini-panels, no panels inside panels.\n\n")

	// --- CHARACTER MASTER REFERENCES（順序を安定化：fileIdx昇順） ---
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

	// big panel はランダムだと事故るので、ここでは "奇数なら最終だけ横長" で固定
	numPanels := len(panels)
	bigPanelIndex := -1
	if numPanels > 1 && numPanels%2 == 1 {
		bigPanelIndex = numPanels - 1
	} else if numPanels > 0 {
		_ = rand.IntN(numPanels)
	}

	us.WriteString("## PANEL BREAKDOWN (STRICT COUNT + FIXED LAYOUT)\n")
	for i, panel := range panels {
		panelNum := i + 1

		panelSize := "Standard"
		if i == bigPanelIndex {
			panelSize = "FULL-WIDTH IMPACT (spans both columns)"
		}

		status := ""
		if i == numPanels-1 {
			status = " - FINAL PANEL"
		}
		us.WriteString(fmt.Sprintf("### PANEL %d [%s]%s\n", panelNum, panelSize, status))

		// 位置を明示（モデルの誤解を減らす）
		if i == bigPanelIndex {
			us.WriteString("- POSITION: Bottom row, full width.\n")
		} else {
			row := (i / 2) + 1
			col := "RIGHT"
			if i%2 == 1 {
				col = "LEFT"
			}
			us.WriteString(fmt.Sprintf("- POSITION: Row %d, %s column.\n", row, col))
		}

		displayName := panel.SpeakerID
		charFileIdx := -1
		if char := pb.characterMap.GetCharacter(panel.SpeakerID); char != nil {
			displayName = char.Name
			if idx, ok := rm.CharacterFiles[char.ID]; ok {
				charFileIdx = idx
			}
		}

		sceneDescription := sanitizeInline(panel.VisualAnchor)
		// 置換は必要なら残す（IDが本文に混ざる場合は事故りやすいので、将来的にはプレースホルダ推奨）
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

		if panel.ReferenceURL != "" {
			if fileIdx, ok := rm.PanelFiles[panel.ReferenceURL]; ok {
				us.WriteString(fmt.Sprintf("- POSE_REF: Use input_file_%d for BODY/POSE/ANATOMY only. IGNORE face/hair/colors from this file.\n", fileIdx))
				if charFileIdx != -1 {
					us.WriteString(fmt.Sprintf("- IDENTITY_FIX: Face/hair/eyes MUST match input_file_%d exactly.\n", charFileIdx))
				}
			}
		}

		if panel.Dialogue != "" {
			cleanText := formatDialogue(panel.Dialogue)

			us.WriteString(fmt.Sprintf("- SPEECH: Speech bubble for [%s].\n", displayName))
			us.WriteString(fmt.Sprintf("  - TEXT_TO_RENDER: \"%s\"\n", cleanText))
			// フォントと正確性
			us.WriteString("  - TYPOGRAPHY: Use professional Japanese manga font (Gothic or Mincho style).\n")
			us.WriteString("  - LANGUAGE: Japanese characters. Ensure each Kanji/Kana is rendered accurately and legibly.\n")
			us.WriteString("  - STYLE: High-quality typesetting. No digital noise inside the text.\n")
		}

		if i == numPanels-1 {
			us.WriteString("- STOP: End of page. Do not draw any additional panels/frames after this.\n")
		}
		us.WriteString("\n")
	}

	userPrompt = us.String()
	return userPrompt, systemPrompt
}
