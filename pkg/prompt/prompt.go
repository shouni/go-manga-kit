package prompt

import (
	"fmt"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"
)

// PromptBuilder は、キャラクター情報を考慮してAIプロンプトを構築します。
type PromptBuilder struct {
	characterMap  domain.CharactersMap
	defaultSuffix string // "anime style, high quality" 等の共通サフィックス
}

// NewPromptBuilder は新しい PromptBuilder を生成します。
func NewPromptBuilder(chars domain.CharactersMap, suffix string) *PromptBuilder {
	return &PromptBuilder{
		characterMap:  chars,
		defaultSuffix: suffix,
	}
}

// BuildFullPagePrompt は、全パネルの情報を統合し、1枚のマンガページを生成するためのプロンプトを構築します。
func (pb *PromptBuilder) BuildFullPagePrompt(mangaTitle string, pages []domain.MangaPage, chars []domain.Character) string {
	var sb strings.Builder

	// 1. 全体構造の定義（MangaStructureHeaderなどは外部定数を想定）
	sb.WriteString("### MANGA PAGE STRUCTURE ###\n")
	sb.WriteString(fmt.Sprintf("- TOTAL PANELS: This page MUST contain exactly %d distinct panels.\n", len(pages)))

	// 2. タイトルと共通スタイルの適用
	sb.WriteString(fmt.Sprintf("\n### TITLE: %s ###\n", mangaTitle))
	if pb.defaultSuffix != "" {
		sb.WriteString(fmt.Sprintf("- GLOBAL_STYLE: %s\n", pb.defaultSuffix))
	}
	sb.WriteString("\n")

	// 3. 登場キャラクターの定義セクション
	sb.WriteString("### CHARACTER IDENTITIES ###\n")
	for _, char := range chars {
		cues := strings.Join(char.VisualCues, ", ")
		sb.WriteString(fmt.Sprintf("- %s: %s\n", char.Name, cues))
	}
	sb.WriteString("\n")

	// 4. 各パネルの具体的な内容を結合
	for i, page := range pages {
		isBig := i == 0 // 最初のコマを大ゴマとする
		panelType := "NORMAL"
		if isBig {
			panelType = "LARGE/ESTABLISHING"
		}
		sb.WriteString(fmt.Sprintf("#### PANEL %d [%s] ####\n", i+1, panelType))
		sb.WriteString(fmt.Sprintf("VISUAL: %s\n", page.VisualAnchor))
		sb.WriteString(fmt.Sprintf("DIALOGUE: %s\n", page.Dialogue))
		sb.WriteString("\n")
	}

	return sb.String()
}

// BuildUnifiedPrompt は、特定のパネル情報とキャラクター情報を統合した単体プロンプトを生成します。
// SpeakerIDをキーにキャラクター設定を引き当て、シード値と合成されたプロンプトを返却するのだ。
func (pb *PromptBuilder) BuildUnifiedPrompt(page domain.MangaPage, speakerID string) (string, int64) {
	var parts []string
	var seed int64

	// 1. キャラクター設定の注入
	if char, ok := pb.characterMap[speakerID]; ok {
		// スライス形式の VisualCues を結合して追加するのだ
		if len(char.VisualCues) > 0 {
			parts = append(parts, char.VisualCues...)
		}
		// キャラクター固有のシードを採用
		seed = char.Seed
	} else if speakerID != "" {
		// 登録がない場合は名前から決定論的なシードを生成するのだ
		seed = int64(domain.GetSeedFromName(speakerID))
	}

	// 2. アクション/ビジュアルアンカー（そのコマ固有の指示）の追加
	if page.VisualAnchor != "" {
		parts = append(parts, page.VisualAnchor)
	}

	// 3. デフォルトサフィックス（画風）の結合
	if pb.defaultSuffix != "" {
		parts = append(parts, pb.defaultSuffix)
	}

	// 空文字を除去してカンマ区切りで結合
	var cleanParts []string
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			cleanParts = append(cleanParts, s)
		}
	}

	return strings.Join(cleanParts, ", "), seed
}
