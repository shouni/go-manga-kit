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
	NegativePagePrompt = "monochrome, black and white, greyscale, screentone, hatching, dot shades, ink sketch, line art only, realistic photos, 3d render, watermark, text, signature, deformed faces, bad anatomy, disfigured, poorly drawn hands"

	MangaStructureHeader = `### FORMAT RULES: FULL COLOR ANIME MANGA ###
- STYLE: Vibrant Full Color Digital Anime Style. High saturation, cinematic lighting.
- RENDERING: Sharp clean lineart with professional digital coloring. NO screentones, NO monochrome hatching.
- COLORING: Full palette, vivid saturation, consistent with character reference sheets.
- LAYOUT: Strict multi-panel composition. NO merging panels.
- BORDERS: Deep black, crisp frame borders for EVERY panel.
- GUTTERS: Pure white space between panels.
- READING FLOW: Right-to-Left, Top-to-Bottom.`
)

// BuildMangaPagePrompt は、ResourceMap を使用して「フルカラー・キャラ一貫性」を両立したプロンプトを構築します。
func (pb *ImagePromptBuilder) BuildMangaPagePrompt(panels []domain.Panel, rm *ResourceMap) (userPrompt string, systemPrompt string) {
	// --- 1. System Prompt (カラーアニメ・役割の固定) ---
	const mangaSystemInstruction = "You are a master digital artist specialized in full-color anime manga. You produce professional, high-saturation pages with cinematic lighting. You must ignore any monochrome elements in references and convert everything to vibrant digital color."

	systemParts := []string{
		mangaSystemInstruction,
		MangaStructureHeader, // 定義済みのカラー版ヘッダー
		RenderingStyle,       // 定義済みのカラー版スタイル
		CinematicTags,        // 定義済みの共通タグ
	}
	if pb.defaultSuffix != "" {
		systemParts = append(systemParts, fmt.Sprintf("### ARTISTIC STYLE ###\n%s", pb.defaultSuffix))
	}
	systemPrompt = strings.Join(systemParts, "\n\n")

	// --- 2. User Prompt (フルカラーページ固有の指示) ---
	var us strings.Builder

	us.WriteString("# FULL COLOR PAGE PRODUCTION REQUEST\n")
	us.WriteString("- OUTPUT TYPE: STRICTLY VIBRANT FULL COLOR.\n")
	us.WriteString(fmt.Sprintf("- PANEL COUNT: Exactly %d distinct panels.\n\n", len(panels)))

	// キャラクター定義 (立ち絵のカラーパレットを正とする)
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
		// 「色」を input_file_N から引き継ぐことを明示
		us.WriteString(fmt.Sprintf("- SUBJECT [%s]: Face and Color MUST follow input_file_%d. (Traits: %s)\n", displayName, fileIdx, visualDesc))
	}
	us.WriteString("\n")

	// 大ゴマの決定
	numPanels := len(panels)
	bigPanelIndex := -1
	if numPanels > 0 {
		bigPanelIndex = rand.IntN(numPanels)
	}

	us.WriteString("## PANEL BREAKDOWN (EXECUTE IN COLOR)\n")
	for i, panel := range panels {
		panelNum := i + 1
		panelSize := "Standard"
		if i == bigPanelIndex {
			panelSize = "LARGE IMPACT"
		}

		us.WriteString(fmt.Sprintf("### PANEL %d [%s]\n", panelNum, panelSize))

		displayName := panel.SpeakerID
		charFileIdx := -1
		if char := pb.characterMap.GetCharacter(panel.SpeakerID); char != nil {
			displayName = char.Name
			charFileIdx = rm.CharacterFiles[char.ID]
		}

		sceneDescription := strings.ReplaceAll(panel.VisualAnchor, panel.SpeakerID, displayName)
		// キャラクターの特徴（VisualCues）を取得
		charCues := ""
		if char := pb.characterMap.GetCharacter(panel.SpeakerID); char != nil && len(char.VisualCues) > 0 {
			charCues = fmt.Sprintf(" (Visual identity from input_file_%d: %s)", rm.CharacterFiles[panel.SpeakerID], strings.Join(char.VisualCues, ", "))
		}

		// プロンプト書き出し
		us.WriteString("- RENDER: FULL COLOR with Rich Saturation.\n")
		us.WriteString(fmt.Sprintf("- SUBJECT: %s\n", displayName))
		// ACTION 1行に集約
		us.WriteString(fmt.Sprintf("- ACTION: %s%s\n", sceneDescription, charCues))

		if panel.ReferenceURL != "" {
			if fileIdx, ok := rm.PanelFiles[panel.ReferenceURL]; ok {
				us.WriteString(fmt.Sprintf("- POSE_REF: Use input_file_%d for BODY ANATOMY ONLY. IGNORE colors/textures from this file.\n", fileIdx))
				if charFileIdx != -1 {
					us.WriteString(fmt.Sprintf("- IDENTITY_FIX: Force [%s]'s hair, eye color, and face to match input_file_%d exactly.\n", displayName, charFileIdx))
				}
			}
		}

		// セリフ指示
		if panel.Dialogue != "" {
			us.WriteString(fmt.Sprintf("- SPEECH: Render a clear bubble for [%s] with text: \"%s\"\n", displayName, panel.Dialogue))
		}
		us.WriteString("\n")
	}

	userPrompt = us.String()
	return userPrompt, systemPrompt
}
