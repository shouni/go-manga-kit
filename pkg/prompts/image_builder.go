package prompts

import (
	"github.com/shouni/go-manga-kit/pkg/domain"
)

// ImagePromptBuilder は、キャラクター情報を考慮してAIプロンプトを構築します。
type ImagePromptBuilder struct {
	characterMap  domain.CharactersMap
	defaultSuffix string // 例: "anime style, high quality"
}

// NewImagePromptBuilder は新しい PromptBuilder を生成します。
func NewImagePromptBuilder(characterMap domain.CharactersMap, suffix string) *ImagePromptBuilder {
	return &ImagePromptBuilder{
		characterMap:  characterMap,
		defaultSuffix: suffix,
	}
}
