package prompts

// TemplateData はレビュープロンプトのテンプレートに渡すデータ構造です。
type TemplateData struct {
	InputText string
}

// ResourceMap は、文字やパネルのリソースファイルをインデックスや順序付きの参照にマッピングするための構造体です。
type ResourceMap struct {
	CharacterFiles map[string]int
	PanelFiles     map[string]int
	OrderedURIs    []string
	OrderedURLs    []string
}
