package prompts

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"
)

// BuildMangaPagePrompt は、UserPrompt（具体的内容）と SystemPrompt（構造・画風）を分けて生成します。
func (pb *ImagePromptBuilder) BuildMangaPagePrompt(panels []domain.Panel, refURLs []string) (userPrompt string, systemPrompt string) {
	return pb.promptVer1(panels, refURLs)
}

// promptVer1 は、UserPrompt（具体的内容）と SystemPrompt（構造・画風）を分けて生成します。
func (pb *ImagePromptBuilder) promptVer1(panels []domain.Panel, refURLs []string) (userPrompt string, systemPrompt string) {
	// --- 1. System Prompt の構築 (AIの役割・画風・基本構造) ---
	const mangaSystemInstruction = "You are a professional manga artist. Create a multi-panel layout. "

	systemParts := []string{
		mangaSystemInstruction,
		MangaStructureHeader,
		RenderingStyle,
		CinematicTags,
	}
	if pb.defaultSuffix != "" {
		styleDNA := fmt.Sprintf("### GLOBAL VISUAL STYLE ###\n%s", pb.defaultSuffix)
		systemParts = append(systemParts, styleDNA)
	}
	systemPrompt = strings.Join(systemParts, "\n\n")

	// --- 2. User Prompt の構築 (具体的なページの内容) ---
	var us strings.Builder
	us.WriteString(fmt.Sprintf("- TOTAL PANELS: Generate exactly %d distinct panels on this single page.\n", len(panels)))

	// キャラクター定義セクション
	us.WriteString(BuildCharacterIdentitySection(pb.characterMap))

	// 大ゴマの決定
	numPanels := len(panels)
	bigPanelIndex := -1
	if numPanels > 0 {
		bigPanelIndex = rand.IntN(numPanels)
	}

	// 各パネルの指示を構築
	for i, panel := range panels {
		panelNum := i + 1
		isBig := (i == bigPanelIndex)

		us.WriteString(BuildPanelHeader(panelNum, numPanels, isBig))

		// 参照指示 (posing and layout)
		if i < len(refURLs) {
			us.WriteString(fmt.Sprintf("- REFERENCE: Use input_file_%d for visual guidance on posing and layout.\n", panelNum))
		}

		// --- キャラクター解決と名前の正規化 ---
		displayName := panel.SpeakerID
		if char := pb.characterMap.GetCharacter(panel.SpeakerID); char != nil {
			displayName = char.Name
		}

		sceneDescription := strings.ReplaceAll(panel.VisualAnchor, panel.SpeakerID, displayName)

		us.WriteString(fmt.Sprintf("- ACTION/SCENE: %s\n", sceneDescription))
		if panel.Dialogue != "" {
			us.WriteString(fmt.Sprintf("- DIALOGUE_CONTEXT: [%s] says \"%s\"\n", displayName, panel.Dialogue))
		}
		us.WriteString("\n")
	}
	userPrompt = us.String()

	return userPrompt, systemPrompt
}
