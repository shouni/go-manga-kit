package generator

import (
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

// BuildUnifiedPrompt は、パネル情報とキャラクターDNAを統合したプロンプトを生成します。
func (pb *PromptBuilder) BuildUnifiedPrompt(page domain.MangaPage) (string, int32) {
	parts := make([]string, 0, 3)
	seed := int32(0)

	// 1. キャラクターDNAの注入
	if dna, ok := pb.dnaMap[page.SpeakerID]; ok {
		if dna.VisualCue != "" {
			parts = append(parts, dna.VisualCue)
		}
		seed = dna.Seed
	} else if page.SpeakerID != "" {
		// 未登録キャラクターでも名前からシードを固定
		seed = GetSeedFromName(page.SpeakerID)
	}

	// 2. アクション/ビジュアルアンカーの追加
	if page.VisualAnchor != "" {
		parts = append(parts, page.VisualAnchor)
	}

	// 3. デフォルトサフィックスの結合
	if pb.defaultSuffix != "" {
		parts = append(parts, pb.defaultSuffix)
	}

	return strings.Join(parts, ", "), seed
}
