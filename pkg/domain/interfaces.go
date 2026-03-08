package domain

// TemplateData はレビュープロンプトのテンプレートに渡すデータ構造です。
type TemplateData struct {
	InputText string
}

// ScriptPrompt は、AIプロンプトを構築する契約です。
type ScriptPrompt interface {
	// Build は、指定されたモード（例: "summary", "character_dialogue"）とデータに基づいてプロンプト文字列を生成します。
	Build(mode string, data TemplateData) (string, error)
}

// ImagePrompt は、AIプロンプトを構築する契約です。
type ImagePrompt interface {
	// BuildPanel は、単一の漫画パネル用のユーザープロンプト、システムプロンプト、および使用するseed値を決定します。
	BuildPanel(panel Panel, char *Character) (userPrompt string, systemPrompt string)
	// BuildPage は、統合された漫画ページ画像用のユーザープロンプトと システムプロンプトを生成します。
	BuildPage(panels []Panel, rm *ResourceMap) (userPrompt string, systemPrompt string)
}
