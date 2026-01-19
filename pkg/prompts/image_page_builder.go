package prompts

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"
)

type ResourceMap struct {
	CharacterFiles map[string]int
	PanelFiles     map[string]int
	OrderedURIs    []string
	OrderedURLs    []string
}

const (
	// NegativePagePrompt コマ数が勝手に増えるのを防ぐワード（extra panels 等）を追加
	NegativePagePrompt = "monochrome, black and white, greyscale, screentone, hatching, dot shades, ink sketch, line art only, realistic photos, 3d render, watermark, text, signature, deformed faces, bad anatomy, disfigured, poorly drawn hands, extra panels, unexpected panels, more than specified panels, split panels"

	// MangaStructureHeader レイアウトの厳格性を強調
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
	// --- 1. System Prompt ---
	const mangaSystemInstruction = "You are a master digital artist. You follow the exact panel count requested. You never add extra panels beyond what is instructed."

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

	// --- 2. User Prompt ---
	var us strings.Builder

	// コマ数指示を「STRICTLY ONLY」で強調
	us.WriteString("# FULL COLOR PAGE PRODUCTION REQUEST\n")
	us.WriteString("- OUTPUT TYPE: STRICTLY VIBRANT FULL COLOR.\n")
	us.WriteString(fmt.Sprintf("- PANEL COUNT: [ %d ] (STRICTLY ONLY %d PANELS. DO NOT ADD ANY MORE).\n\n", len(panels), len(panels)))

	// キャラクター定義
	us.WriteString("## CHARACTER MASTER REFERENCES (FIXED COLOR PALETTE)\n")
	for sID, fileIdx := range rm.CharacterFiles {
		displayName := sID
		visualDesc := "vivid anime color palette"

		if char := pb.characterMap.GetCharacter(sID); char != nil {
			displayName = char.Name
			if len(char.VisualCues) > 0 {
				visualDesc = strings.Join(char.VisualCues, ", ")
			}
		}
		us.WriteString(fmt.Sprintf("- SUBJECT [%s]: Face and Color MUST follow input_file_%d. (Traits: %s)\n", displayName, fileIdx, visualDesc))
	}
	us.WriteString("\n")

	numPanels := len(panels)
	bigPanelIndex := -1
	if numPanels > 0 {
		bigPanelIndex = rand.IntN(numPanels)
	}

	us.WriteString("## PANEL BREAKDOWN (STRICT COUNT EXECUTION)\n")
	for i, panel := range panels {
		panelNum := i + 1
		panelSize := "Standard"
		if i == bigPanelIndex {
			panelSize = "LARGE IMPACT"
		}

		// パネルヘッダーに「これが何枚目か」と「最後かどうか」を明示
		status := ""
		if i == numPanels-1 {
			status = " - FINAL PANEL"
		}
		us.WriteString(fmt.Sprintf("### PANEL %d [%s]%s\n", panelNum, panelSize, status))

		displayName := panel.SpeakerID
		charFileIdx := -1
		if char := pb.characterMap.GetCharacter(panel.SpeakerID); char != nil {
			displayName = char.Name
			charFileIdx = rm.CharacterFiles[char.ID]
		}

		sceneDescription := strings.ReplaceAll(panel.VisualAnchor, panel.SpeakerID, displayName)
		charCues := ""
		if char := pb.characterMap.GetCharacter(panel.SpeakerID); char != nil && len(char.VisualCues) > 0 {
			charCues = fmt.Sprintf(" (Visual identity from input_file_%d: %s)", rm.CharacterFiles[panel.SpeakerID], strings.Join(char.VisualCues, ", "))
		}

		us.WriteString("- RENDER: FULL COLOR.\n")
		us.WriteString(fmt.Sprintf("- SUBJECT: %s\n", displayName))
		us.WriteString(fmt.Sprintf("- ACTION: %s%s\n", sceneDescription, charCues))

		if panel.ReferenceURL != "" {
			if fileIdx, ok := rm.PanelFiles[panel.ReferenceURL]; ok {
				us.WriteString(fmt.Sprintf("- POSE_REF: Use input_file_%d for BODY ANATOMY ONLY. IGNORE colors/face from this file.\n", fileIdx))
				if charFileIdx != -1 {
					us.WriteString(fmt.Sprintf("- IDENTITY_FIX: Force [%s]'s hair, eyes, and face to match input_file_%d exactly.\n", displayName, charFileIdx))
				}
			}
		}

		if panel.Dialogue != "" {
			us.WriteString(fmt.Sprintf("- SPEECH: Render a clear bubble for [%s] with text: \"%s\"\n", displayName, panel.Dialogue))
		}

		// 最後のパネルの後に停止を命じる
		if i == numPanels-1 {
			us.WriteString("- STOP: This is the end of the page. Do not draw any more content after this frame.\n")
		}
		us.WriteString("\n")
	}

	userPrompt = us.String()
	return userPrompt, systemPrompt
}
