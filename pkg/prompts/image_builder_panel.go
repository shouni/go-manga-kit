package prompts

import (
	"fmt"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"
)

const (
	// NegativePanelPrompt 単体パネルでは「文字」や「フキダシ」を徹底排除します
	NegativePanelPrompt = "speech bubble, dialogue balloon, text, alphabet, letters, words, signatures, watermark, username, low quality, distorted, bad anatomy, monochrome, black and white, greyscale"
)

// BuildPanel は、単体パネル用の UserPrompt, SystemPrompt, およびシード値を生成します。
func (pb *ImagePromptBuilder) BuildPanel(panel domain.Panel, speakerID string) (userPrompt string, systemPrompt string, targetSeed int64) {
	// --- 1. System Prompt の構築 ---
	const mangaSystemInstruction = "You are a professional anime illustrator. Create a single high-quality cinematic scene with vibrant digital coloring."

	systemParts := []string{
		mangaSystemInstruction,
		RenderingStyle,
		CinematicTags,
	}
	if pb.defaultSuffix != "" {
		styleDNA := fmt.Sprintf("### ARTISTIC STYLE ###\n%s", pb.defaultSuffix)
		systemParts = append(systemParts, styleDNA)
	}
	systemPrompt = strings.Join(systemParts, "\n\n")

	// --- 2. キャラクター設定とアクションの構築 (User Prompt) ---
	var visualParts []string
	displayName := speakerID

	// キャラクターの特定
	char := pb.characterMap.GetCharacterWithDefault(speakerID)
	if char != nil {
		displayName = char.Name
		targetSeed = char.Seed

		// 1. まずキャラクターの固有名詞を主語として入れる
		visualParts = append(visualParts, fmt.Sprintf("Subject: %s", displayName))

		// 2. キャラクター固有のビジュアル特徴(VisualCues)を追加
		if len(char.VisualCues) > 0 {
			visualParts = append(visualParts, char.VisualCues...)
		}
	} else {
		// キャラクターが見つからない場合のフォールバック
		targetSeed = domain.GetSeedFromString(speakerID)
		visualParts = append(visualParts, displayName)
	}

	// 3. アクション/シーン描写(VisualAnchor)を追加
	// IDを名前に置換して、文脈を明確にする
	if panel.VisualAnchor != "" {
		actionDesc := strings.ReplaceAll(panel.VisualAnchor, speakerID, displayName)
		visualParts = append(visualParts, fmt.Sprintf("Action: %s", actionDesc))
	}

	// 4. カラーと品質の最終念押し
	visualParts = append(visualParts, "vibrant full color", "cinematic lighting")

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
