package prompts

import mangadom "github.com/shouni/go-manga-kit/pkg/domain"

// ScriptPrompt は、AIプロンプトを構築する契約です。
type ScriptPrompt interface {
	// Build は、指定されたモード（例: "summary", "character_dialogue"）とデータに基づいてプロンプト文字列を生成します。
	Build(mode string, data TemplateData) (string, error)
}

// ImagePrompt は、AIプロンプトを構築する契約です。
type ImagePrompt interface {
	// BuildPanel は、単一の漫画パネル用のユーザープロンプト、システムプロンプト、および使用するseed値を決定します。
	BuildPanel(panel mangadom.Panel, speakerID string) (userPrompt string, systemPrompt string, targetSeed int64)
	// BuildPage は、統合された漫画ページ画像用のユーザープロンプトと システムプロンプトを生成します。
	BuildPage(panels []mangadom.Panel, rm *ResourceMap) (userPrompt string, systemPrompt string)
}
