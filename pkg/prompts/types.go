package prompts

import imgdom "github.com/shouni/gemini-image-kit/pkg/domain"

// TemplateData はレビュープロンプトのテンプレートに渡すデータ構造です。
type TemplateData struct {
	InputText string
}

// ResourceMap は、文字やパネルのリソースファイルをインデックスや順序付きの参照にマッピングするための構造体です。
type ResourceMap struct {
	// CharacterFiles は SpeakerID から OrderedAssets のインデックスへのマップです。
	CharacterFiles map[string]int
	// PanelFiles は ReferenceURL から OrderedAssets のインデックスへのマップです。
	PanelFiles map[string]int
	// OrderedAssets は Gemini に渡す画像アセット（File API URI と元の URL のペア）の順序付きリストです。
	OrderedAssets []imgdom.ImageURI
}
