package prompts

// TemplateData はレビュープロンプトのテンプレートに渡すデータ構造です。
type TemplateData struct {
	InputText string
}

// ResourceMap 文字およびパネル リソース ファイルを特定のインデックスおよび順序付き参照にマップするために使用される構造です
type ResourceMap struct {
	CharacterFiles map[string]int
	PanelFiles     map[string]int
	OrderedURIs    []string
	OrderedURLs    []string
}
