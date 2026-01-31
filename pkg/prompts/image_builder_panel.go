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

	// キャラクター情報を取得
	speakerID := panel.SpeakerID
	displayName := char.Name

	// 編集者AIが生成した VisualAnchorをベースにする
	if panel.VisualAnchor != "" {
		anchor := strings.ReplaceAll(panel.VisualAnchor, speakerID, displayName)
		visualParts = append(visualParts, anchor)
	} else {
		// 万が一 Anchor が空の場合のフォールバック
		visualParts = append(visualParts, fmt.Sprintf("%s character, character focus", displayName))
	}

	// キャラクター固有のビジュアル特徴 (VisualCues) を追加
	if len(char.VisualCues) > 0 {
		visualParts = append(visualParts, char.VisualCues...)
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
