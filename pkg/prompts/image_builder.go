package prompts

import (
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
