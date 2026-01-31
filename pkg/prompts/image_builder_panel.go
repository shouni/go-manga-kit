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
func (pb *ImagePromptBuilder) BuildPanel(panel domain.Panel, char *domain.Character) (userPrompt string, systemPrompt string) {
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

	// --- 2. User Prompt の構築 ---
	var visualParts []string
	speakerID := panel.SpeakerID
	displayName := char.Name

	// アイデンティティの固定を最優先
	// プロンプトの先頭に「名前 + 外見的特徴」を置くことで、キャラクターの再現度を最大化します。
	identityBase := fmt.Sprintf("%s character", displayName)
	if len(char.VisualCues) > 0 {
		identityBase = fmt.Sprintf("%s, %s", identityBase, strings.Join(char.VisualCues, ", "))
	}
	visualParts = append(visualParts, identityBase)

	// 編集者AIが生成した VisualAnchor (アクション・構図・背景)
	// IDを表示名に置換して結合します。
	if panel.VisualAnchor != "" {
		anchor := strings.ReplaceAll(panel.VisualAnchor, speakerID, displayName)
		visualParts = append(visualParts, anchor)
	} else {
		// フォールバック
		visualParts = append(visualParts, "character focus, cinematic scene")
	}

	visualParts = append(visualParts, "vibrant full color", "cinematic lighting", "high quality")

	// --- 3. プロンプトのクリーンな結合 ---
	var cleanParts []string
	for _, p := range visualParts {
		if s := strings.TrimSpace(p); s != "" {
			cleanParts = append(cleanParts, s)
		}
	}
	userPrompt = strings.Join(cleanParts, ", ")

	return userPrompt, systemPrompt
}
