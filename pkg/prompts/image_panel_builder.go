package prompts

import (
	"fmt"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"
)

const (
	// NegativePanelPrompt は、パネル用のネガティブプロンプトです。
	NegativePanelPrompt = "speech bubble, dialogue balloon, text, alphabet, letters, words, signatures, watermark, username, low quality, distorted, bad anatomy"
)

// BuildPanelPrompt は、単体パネル用の UserPrompt, SystemPrompt, およびシード値を生成します。
func (pb *ImagePromptBuilder) BuildPanelPrompt(panel domain.Panel, speakerID string) (userPrompt string, systemPrompt string, targetSeed int64) {
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
	systemPrompt = strings.Join(systemParts, "\n\n")

	// --- 2. キャラクター設定とビジュアルアンカーの収集 (User Prompt) ---
	var visualParts []string

	// キャラクターの特定とフォールバック処理
	char := pb.characterMap.GetCharacterWithDefault(speakerID)
	// キャラクター（またはPrimary）が見つかった場合の処理
	if char != nil {
		if len(char.VisualCues) > 0 {
			visualParts = append(visualParts, char.VisualCues...)
		}
		targetSeed = char.Seed
	} else {
		targetSeed = domain.GetSeedFromString(speakerID)
	}

	// アクション/ビジュアルアンカーの追加
	if panel.VisualAnchor != "" {
		visualParts = append(visualParts, panel.VisualAnchor)
	}

	// --- 3. プロンプトのクリーンな結合 ---
	var cleanParts []string
	for _, p := range visualParts {
		if s := strings.TrimSpace(p); s != "" {
			cleanParts = append(cleanParts, s)
		}
	}
	userPrompt = strings.Join(cleanParts, ", ")

	return userPrompt, systemPrompt, targetSeed
}
