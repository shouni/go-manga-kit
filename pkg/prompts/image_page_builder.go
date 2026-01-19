package prompts

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"
)

// ResourceMap は、AIに送信するファイルとプロンプト上のインデックスの対応を管理します。
type ResourceMap struct {
	CharacterFiles map[string]int // SpeakerID -> input_file_N のインデックス
	PanelFiles     map[string]int // ReferenceURL -> input_file_N のインデックス
	OrderedURIs    []string       // 最終的に File API に送る URI のリスト
	OrderedURLs    []string       // 最終的に ReferenceURLs に入れる URL のリスト
}

const (
	// NegativePagePrompt は、ページ用のネガティブプロンプトです。
	NegativePagePrompt = "deformed faces, mismatched eyes, cross-eyed, low-quality faces, blurry facial features, melting faces, extra limbs, merged panels, messy lineart, distorted anatomy"
	//NegativePagePrompt = "deformed faces, low-quality faces, blurry features, extra limbs, distorted anatomy, messy lineart"

	// MangaStructureHeader は作画の全体構造を定義します。
	MangaStructureHeader = `### MANDATORY FORMAT: MULTI-PANEL MANGA PAGE COMPOSITION ###
	- STRUCTURE: A professional Japanese manga spread with clear frame borders.
	- READING ORDER: Right-to-Left, Top-to-Bottom.
	- GUTTERS: Ultra-thin, crisp hairline dividers. NO OVERLAPPING. Each panel is a separate scene.`

	//	MangaStructureHeader = `### MANDATORY FORMAT: MULTI-PANEL MANGA PAGE ###
	//- FRAME BORDERS: Every panel must have solid, visible black borders.
	//- GUTTERS: Use clear white spacing between panels. NO overlapping or merging.
	//- LAYOUT: Strictly separate each numbered panel into its own spatial area.`
)

// BuildMangaPagePrompt は、ResourceMap を使用して高精度なプロンプトを構築します。
func (pb *ImagePromptBuilder) BuildMangaPagePrompt(panels []domain.Panel, rm *ResourceMap) (userPrompt string, systemPrompt string) {
	// --- 1. System Prompt (画風・構造の強制) ---
	const mangaSystemInstruction = "You are a professional manga artist. Create a multi-panel layout. "
	systemParts := []string{
		mangaSystemInstruction,
		MangaStructureHeader,
		RenderingStyle,
		CinematicTags,
	}
	if pb.defaultSuffix != "" {
		systemParts = append(systemParts, fmt.Sprintf("### GLOBAL VISUAL STYLE ###\n%s", pb.defaultSuffix))
	}
	systemPrompt = strings.Join(systemParts, "\n\n")

	// --- 2. User Prompt (ページ固有の指示) ---
	var us strings.Builder
	us.WriteString(fmt.Sprintf("- TOTAL PANELS: Generate exactly %d distinct panels.\n", len(panels)))

	// [重要] キャラクターの見た目定義セクション
	us.WriteString("### CHARACTER VISUAL MASTER DEFINITIONS ###\n")
	for sID, fileIdx := range rm.CharacterFiles {
		displayName := sID
		if char := pb.characterMap.GetCharacter(sID); char != nil {
			displayName = char.Name
			// ビジュアル情報を付加
			cues := strings.Join(char.VisualCues, ", ")
			us.WriteString(fmt.Sprintf("- SUBJECT [%s]: Follow identity in input_file_%d. Visual features: {%s}\n", displayName, fileIdx, cues))
		} else {
			us.WriteString(fmt.Sprintf("- SUBJECT [%s]: Follow identity in input_file_%d.\n", displayName, fileIdx))
		}
	}
	us.WriteString("\n")

	// 大ゴマの決定
	numPanels := len(panels)
	bigPanelIndex := -1
	if numPanels > 0 {
		bigPanelIndex = rand.IntN(numPanels)
	}

	for i, panel := range panels {
		panelNum := i + 1
		isBig := (i == bigPanelIndex)

		// パネルヘッダー
		us.WriteString(BuildPanelHeader(panelNum, numPanels, isBig))
		us.WriteString("- FRAME_RULE: Keep all content strictly inside black borders. NO overlapping.\n")

		// キャラクター名の正規化
		displayName := panel.SpeakerID
		if char := pb.characterMap.GetCharacter(panel.SpeakerID); char != nil {
			displayName = char.Name
		}

		// [改善点] セリフの描画指示 (優先順位：高)
		if panel.Dialogue != "" {
			us.WriteString(fmt.Sprintf("- SPEECH_BUBBLE: Render a clear dialogue bubble for [%s] with text: \"%s\"\n", displayName, panel.Dialogue))
		}

		// [改善点] ポーズ参照の紐付け
		if panel.ReferenceURL != "" {
			if fileIdx, ok := rm.PanelFiles[panel.ReferenceURL]; ok {
				us.WriteString(fmt.Sprintf("- POSE_REFERENCE: Use input_file_%d for layout/posing. Ensure [%s]'s face matches their character master file.\n", fileIdx, displayName))
			}
		}

		// シーン描写
		sceneDescription := strings.ReplaceAll(panel.VisualAnchor, panel.SpeakerID, displayName)
		us.WriteString(fmt.Sprintf("- ACTION/SCENE: %s\n", sceneDescription))
		us.WriteString("\n")
	}

	userPrompt = us.String()
	return userPrompt, systemPrompt
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
