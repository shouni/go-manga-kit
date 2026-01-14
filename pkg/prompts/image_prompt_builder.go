package prompts

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"
)

// ImagePromptBuilder は、キャラクター情報を考慮してAIプロンプトを構築します。
type ImagePromptBuilder struct {
	characterMap  domain.CharactersMap
	defaultSuffix string // "anime style, high quality" 等の共通サフィックス
}

// NewImagePromptBuilder は新しい PromptBuilder を生成します。
func NewImagePromptBuilder(chars domain.CharactersMap, suffix string) *ImagePromptBuilder {
	return &ImagePromptBuilder{
		characterMap:  chars,
		defaultSuffix: suffix,
	}
}

// BuildUnifiedPrompt は、単体パネル用の UserPrompt, SystemPrompt, およびシード値を生成します。
func (pb *ImagePromptBuilder) BuildUnifiedPrompt(page domain.MangaPage, speakerID string) (string, string, int64) {
	// --- 1. System Prompt の構築 ---
	// 単体パネル生成では、1枚の高品質なイラストとしての役割と画風を定義します。
	var ss strings.Builder
	const mangaSystemInstruction = "You are a professional anime illustrator. Create a single high-quality cinematic scene."
	ss.WriteString(mangaSystemInstruction)
	ss.WriteString("\n\n")
	ss.WriteString(RenderingStyle)
	ss.WriteString("\n\n")
	if pb.defaultSuffix != "" {
		ss.WriteString("\n\n")
		ss.WriteString(fmt.Sprintf("### GLOBAL VISUAL STYLE ###\n%s", pb.defaultSuffix))
	}
	systemPrompt := ss.String()

	// --- 2. キャラクター設定とビジュアルアンカーの収集 (User Prompt) ---
	var visualParts []string
	var targetSeed int64
	if char, ok := pb.characterMap[speakerID]; ok {
		// 登録済みキャラクターの場合、そのDNA（VisualCuesとSeed）を完全に継承します
		if len(char.VisualCues) > 0 {
			visualParts = append(visualParts, char.VisualCues...)
		}
		targetSeed = char.Seed
	} else {
		// 登録がない場合は、名前から決定論的にシード値を生成します
		targetSeed = domain.GetSeedFromName(speakerID, pb.characterMap)

		if speakerID != "" {
			visualParts = append(visualParts, speakerID)
		}
	}

	// アクション/ビジュアルアンカーの追加
	if page.VisualAnchor != "" {
		visualParts = append(visualParts, page.VisualAnchor)
	}

	// クオリティ向上タグの追加
	visualParts = append(visualParts, CinematicTags)

	// --- 3. プロンプトのクリーンな結合 ---
	var cleanParts []string
	for _, p := range visualParts {
		if s := strings.TrimSpace(p); s != "" {
			cleanParts = append(cleanParts, s)
		}
	}
	prompt := strings.Join(cleanParts, ", ")

	return prompt, systemPrompt, targetSeed
}

// BuildFullPagePrompt は、UserPrompt（具体的内容）と SystemPrompt（構造・画風）を分けて生成します。
func (pb *ImagePromptBuilder) BuildFullPagePrompt(mangaTitle string, pages []domain.MangaPage, refURLs []string) (string, string) {
	// --- 1. System Prompt の構築 (AIの役割・画風・基本構造) ---
	var ss strings.Builder
	const mangaSystemInstruction = "You are a professional manga artist. Create a multi-panel layout. "
	ss.WriteString(mangaSystemInstruction)
	ss.WriteString("\n\n")
	ss.WriteString(MangaStructureHeader)
	ss.WriteString("\n\n")
	ss.WriteString(RenderingStyle)
	ss.WriteString("\n\n")
	if pb.defaultSuffix != "" {
		ss.WriteString(fmt.Sprintf("\n- GLOBAL_STYLE_DNA: %s\n", pb.defaultSuffix))
	}
	systemPrompt := ss.String()

	// --- 2. User Prompt の構築 (具体的なページの内容) ---
	var us strings.Builder
	us.WriteString(fmt.Sprintf("### TITLE: %s ###\n", mangaTitle))
	us.WriteString(fmt.Sprintf("- TOTAL PANELS: Generate exactly %d distinct panels on this single page.\n", len(pages)))

	// キャラクター定義セクション
	us.WriteString(BuildCharacterIdentitySection(pb.characterMap))

	// 大ゴマの決定
	numPanels := len(pages)
	bigPanelIndex := -1
	if numPanels > 0 {
		bigPanelIndex = rand.IntN(numPanels)
	}

	// 各パネルの指示
	for i, page := range pages {
		panelNum := i + 1
		isBig := (i == bigPanelIndex)

		us.WriteString(BuildPanelHeader(panelNum, numPanels, isBig))

		// 参照画像のインデックス指定
		if i < len(refURLs) {
			us.WriteString(fmt.Sprintf("- REFERENCE: See input_file_%d for visual guidance.\n", panelNum))
		}

		us.WriteString(fmt.Sprintf("- ACTION/SCENE: %s\n", page.VisualAnchor))
		if page.Dialogue != "" {
			us.WriteString(fmt.Sprintf("- DIALOGUE_CONTEXT: [%s] says \"%s\"\n", page.SpeakerID, page.Dialogue))
		}
		us.WriteString("\n")
	}

	return us.String(), systemPrompt
}
