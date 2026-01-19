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
	// NegativePagePrompt 品質低下を防ぐためのネガティブプロンプトを強化
	NegativePagePrompt = "color, realistic photos, 3d render, watermark, text, signature, sketch, deformed faces, bad anatomy, disfigured, poorly drawn face, mutation, extra limb, ugly, disgusting, poorly drawn hands, missing limb, floating limbs, disconnected limbs, malformed hands, blurry, mutated hands, fingers"

	// MangaStructureHeader マンガの構造（白黒、枠線、余白）をより具体的に定義
	MangaStructureHeader = `### FORMAT RULES: PROFESSIONAL MANGA PAGE ###
- STYLE: High-contrast black and white Japanese Manga. Use G-Pen ink lines and screentones.
- LAYOUT: Strict multi-panel composition. NO merging panels.
- BORDERS: Deep black, crisp frame borders for EVERY panel.
- GUTTERS: Pure white space between panels.
- READING FLOW: Right-to-Left, Top-to-Bottom.`
)

// BuildMangaPagePrompt は、ResourceMap を使用して高精度なプロンプトを構築します。
func (pb *ImagePromptBuilder) BuildMangaPagePrompt(panels []domain.Panel, rm *ResourceMap) (userPrompt string, systemPrompt string) {
	// --- 1. System Prompt (画風・構造の強制・役割付与) ---
	const mangaSystemInstruction = "You are a master manga artist (Mangaka). You excel at dynamic composition, expressive line art, and coherent storytelling."

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

	// --- 2. User Prompt (ページ固有の指示) ---
	var us strings.Builder

	// ページの全体像を先に提示
	us.WriteString(fmt.Sprintf("# PAGE REQUEST\nCreate a strictly segmented manga page with EXACTLY %d panels.\n\n", len(panels)))

	// キャラクター定義 (Reference Sheet)
	// プロンプトの先頭で定義することで一貫性を高める
	us.WriteString("## CHARACTER REFERENCE SHEET\n")
	for sID, fileIdx := range rm.CharacterFiles {
		displayName := sID
		visualDesc := "Distinct character features" // デフォルト

		if char := pb.characterMap.GetCharacter(sID); char != nil {
			displayName = char.Name
			if len(char.VisualCues) > 0 {
				visualDesc = strings.Join(char.VisualCues, ", ")
			}
		}
		// ファイル参照とテキスト記述を強力に結びつける
		us.WriteString(fmt.Sprintf("- REF_ID [%s]: Look at input_file_%d. Traits: {%s}\n", displayName, fileIdx, visualDesc))
	}
	us.WriteString("\n")

	// 大ゴマの決定（ランダム）
	numPanels := len(panels)
	bigPanelIndex := -1
	if numPanels > 0 {
		bigPanelIndex = rand.IntN(numPanels)
	}

	us.WriteString("## PANEL BREAKDOWN\n")
	for i, panel := range panels {
		panelNum := i + 1

		// パネルごとのヘッダー作成
		// 大ゴマか通常かによってカメラワークの重み付けを変える指示を入れる
		panelSize := "Standard Size"
		shotFocus := "Medium Shot" // デフォルト
		if i == bigPanelIndex {
			panelSize = "LARGE IMPACT PANEL"
			shotFocus = "Dynamic Angle / Close-up or Wide Detailed Shot"
		}

		us.WriteString(fmt.Sprintf("### PANEL %d [%s]\n", panelNum, panelSize))

		// キャラクター名の解決
		displayName := panel.SpeakerID
		if char := pb.characterMap.GetCharacter(panel.SpeakerID); char != nil {
			displayName = char.Name
		}

		// シーン描写の構築
		// 単純な置換だけでなく、主語を明確にする
		sceneDescription := strings.ReplaceAll(panel.VisualAnchor, panel.SpeakerID, displayName)

		// 構造化された指示ブロック
		us.WriteString(fmt.Sprintf("- FOCUS: %s\n", shotFocus))
		us.WriteString(fmt.Sprintf("- SUBJECT: %s\n", displayName))
		us.WriteString(fmt.Sprintf("- ACTION: %s\n", sceneDescription))

		// ポーズ参照がある場合
		if panel.ReferenceURL != "" {
			if fileIdx, ok := rm.PanelFiles[panel.ReferenceURL]; ok {
				us.WriteString(fmt.Sprintf("- POSE_REF: Use input_file_%d for composition/anatomy only. Keep face of [%s].\n", fileIdx, displayName))
			}
		}

		// セリフ/フキダシ指示
		// テキスト描画はAIにとって難しいので、明確な引用符と配置指示を与える
		if panel.Dialogue != "" {
			us.WriteString(fmt.Sprintf("- SPEECH: Place a dialogue bubble near [%s]. Text contents: \"%s\"\n", displayName, panel.Dialogue))
		} else {
			us.WriteString("- SPEECH: No dialogue in this panel.\n")
		}

		us.WriteString("\n") // パネル間の空行
	}

	userPrompt = us.String()
	return userPrompt, systemPrompt
}
