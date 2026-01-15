package prompts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
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

// BuildPanelPrompt は、単体パネル用の UserPrompt, SystemPrompt, およびシード値を生成します。
func (pb *ImagePromptBuilder) BuildPanelPrompt(page domain.MangaPage, speakerID string) (string, string, int64) {
	// --- 1. System Prompt の構築 ---
	const mangaSystemInstruction = "You are a professional anime illustrator. Create a single high-quality cinematic scene."

	// CinematicTags を System Prompt に移動し、全体的な画風としての責務を一貫させます
	systemParts := []string{
		mangaSystemInstruction,
		RenderingStyle,
		CinematicTags,
	}
	if pb.defaultSuffix != "" {
		styleDNA := fmt.Sprintf("### GLOBAL VISUAL STYLE ###\n%s", pb.defaultSuffix)
		systemParts = append(systemParts, styleDNA)
	}
	systemPrompt := strings.Join(systemParts, "\n\n")

	// --- 2. キャラクター設定とビジュアルアンカーの収集 (User Prompt) ---
	var visualParts []string
	var targetSeed int64
	if char, ok := pb.characterMap[speakerID]; ok {
		if len(char.VisualCues) > 0 {
			visualParts = append(visualParts, char.VisualCues...)
		}
		targetSeed = char.Seed
	} else {
		targetSeed = domain.GetSeedFromName(speakerID, pb.characterMap)
		if speakerID != "" {
			visualParts = append(visualParts, speakerID)
		}
	}

	// アクション/ビジュアルアンカーの追加
	if page.VisualAnchor != "" {
		visualParts = append(visualParts, page.VisualAnchor)
	}

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

// BuildMangaPagePrompt は、UserPrompt（具体的内容）と SystemPrompt（構造・画風）を分けて生成します。
func (pb *ImagePromptBuilder) BuildMangaPagePrompt(mangaTitle string, pages []domain.MangaPage, refURLs []string) (userPrompt string, systemPrompt string) {
	// --- 1. System Prompt の構築 (AIの役割・画風・基本構造) ---
	var ss strings.Builder
	const mangaSystemInstruction = "You are a professional manga artist. Create a multi-panel layout. "
	ss.WriteString(mangaSystemInstruction)
	ss.WriteString("\n\n")
	ss.WriteString(MangaStructureHeader)
	ss.WriteString("\n\n")
	ss.WriteString(RenderingStyle)
	ss.WriteString("\n\n")
	ss.WriteString(CinematicTags)
	if pb.defaultSuffix != "" {
		ss.WriteString(fmt.Sprintf("\n- GLOBAL_STYLE_DNA: %s\n", pb.defaultSuffix))
	}
	systemPrompt = ss.String()

	// --- 2. User Prompt の構築 (具体的なページの内容) ---
	var us strings.Builder
	// TODO::ページ単位でのタイトルは現時点では出力しない
	//	us.WriteString(fmt.Sprintf("### TITLE: %s ###\n", mangaTitle))
	us.WriteString(fmt.Sprintf("- TOTAL PANELS: Generate exactly %d distinct panels on this single page.\n", len(pages)))

	// キャラクター定義セクション
	us.WriteString(BuildCharacterIdentitySection(pb.characterMap))

	// 大ゴマの決定
	numPanels := len(pages)
	bigPanelIndex := -1
	if numPanels > 0 {
		bigPanelIndex = rand.IntN(numPanels)
	}

	// slog.With を利用した宣言的なロギング
	logger := slog.With(
		"manga_title", mangaTitle,
		"style_suffix", pb.defaultSuffix,
		"panel_count", numPanels,
	)
	if bigPanelIndex != -1 {
		logger = logger.With("big_panel_index", bigPanelIndex)
	}
	logger.Info("Building manga page prompt")

	// 各パネルの指示
	for i, page := range pages {
		panelNum := i + 1
		isBig := (i == bigPanelIndex)

		us.WriteString(BuildPanelHeader(panelNum, numPanels, isBig))

		// 参照指示を具体化: "posing and layout" を明示してAIの精度を向上させます
		if i < len(refURLs) {
			us.WriteString(fmt.Sprintf("- REFERENCE: Use input_file_%d for visual guidance on posing and layout.\n", panelNum))
		}

		// SpeakerID を Name に変換して AI に伝える
		displayName := pb.findCharacterName(page.SpeakerID)

		// アクション指示の中にある SpeakerID も名前に置換して AI の混乱を防ぐ
		sceneDescription := page.VisualAnchor
		if char, ok := pb.characterMap[page.SpeakerID]; ok {
			sceneDescription = strings.ReplaceAll(sceneDescription, page.SpeakerID, char.Name)
		}

		us.WriteString(fmt.Sprintf("- ACTION/SCENE: %s\n", sceneDescription))
		if page.Dialogue != "" {
			us.WriteString(fmt.Sprintf("- DIALOGUE_CONTEXT: [%s] says \"%s\"\n", displayName, page.Dialogue))
		}
		us.WriteString("\n")
	}
	userPrompt = us.String()

	return userPrompt, systemPrompt
}

// findCharacter は SpeakerID からキャラクター名を特定します。
// ID直接一致、または台本形式(speaker-hash)の両方に対応します。
func (pb *ImagePromptBuilder) findCharacterName(speakerID string) string {
	sid := strings.ToLower(speakerID)
	if char, ok := pb.characterMap[sid]; ok {
		return char.Name
	}

	cleanID := strings.TrimPrefix(sid, "speaker-")
	for _, char := range pb.characterMap {
		h := sha256.Sum256([]byte(char.ID))
		hash := hex.EncodeToString(h[:])
		if cleanID == hash[:10] || cleanID == char.ID {
			return char.Name
		}
	}

	return speakerID
}
