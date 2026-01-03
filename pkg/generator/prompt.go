package generator

import (
	"fmt"
	"strings"

	"github.com/shouni/gemini-image-kit/pkg/domain"
)

// PromptBuilder は、キャラクターDNAを考慮してAIプロンプトを構築します。
type PromptBuilder struct {
	dnaMap        DNAMap
	defaultSuffix string // "anime style, high quality" 等の共通サフィックス
}

// NewPromptBuilder は新しい PromptBuilder を生成します。
func NewPromptBuilder(dna DNAMap, suffix string) *PromptBuilder {
	return &PromptBuilder{
		dnaMap:        dna,
		defaultSuffix: suffix,
	}
}

// BuildFullPagePrompt は、全パネルの情報を統合し、1枚のマンガページを生成するための巨大なプロンプトを構築します。
func (pb *PromptBuilder) BuildFullPagePrompt(mangaTitle string, pages []string, dnas []CharacterDNA) string {
	var sb strings.Builder

	// 1. 全体構造の定義
	sb.WriteString(MangaStructureHeader)
	sb.WriteString(fmt.Sprintf("\n- TOTAL PANELS: This page MUST contain exactly %d distinct panels.\n", len(pages)))

	// 2. タイトルと共通スタイルの適用
	sb.WriteString(fmt.Sprintf("\n### TITLE: %s ###\n", mangaTitle))
	sb.WriteString(RenderingStyle)
	if pb.defaultSuffix != "" {
		sb.WriteString(fmt.Sprintf("- STYLE_DNA: %s\n", pb.defaultSuffix))
	}
	sb.WriteString("\n")

	// 3. 登場キャラクターのDNA定義
	sb.WriteString(BuildCharacterIdentitySection(dnas))

	// 4. 各パネルの具体的な内容を結合
	for i, panelPrompt := range pages {
		isBig := i == 0 // 最初のコマを大ゴマとする簡易ロジック
		sb.WriteString(BuildPanelHeader(i+1, len(pages), isBig))
		sb.WriteString(panelPrompt)
		sb.WriteString("\n")
	}

	return sb.String()
}

// BuildUnifiedPrompt は、パネル情報とキャラクターDNAを統合したプロンプトを生成します。
func (pb *PromptBuilder) BuildUnifiedPrompt(page domain.MangaPage) (string, int32) {
	parts := make([]string, 0, 3)
	seed := int32(0)

	// SpeakerIDが存在する場合、まず名前からシードを決定
	if page.SpeakerID != "" {
		seed = GetSeedFromName(page.SpeakerID)
	}

	// 1. 登録済みキャラクターDNAの注入
	if dna, ok := pb.dnaMap[page.SpeakerID]; ok {
		if dna.VisualCue != "" {
			parts = append(parts, dna.VisualCue)
		}
		// 登録済みのシードで上書き（手動指定の可能性があるため）
		seed = dna.Seed
	}

	// 2. アクション/ビジュアルアンカーの追加
	if page.VisualAnchor != "" {
		parts = append(parts, page.VisualAnchor)
	}

	// 3. デフォルトサフィックスの結合
	if pb.defaultSuffix != "" {
		parts = append(parts, pb.defaultSuffix)
	}

	// 空の要素をフィルタリング
	filteredParts := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			filteredParts = append(filteredParts, p)
		}
	}

	return strings.Join(filteredParts, ", "), seed
}
