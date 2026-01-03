package parser

import "regexp"

var (
	// TitleRegex は "# タイトル" 形式のタイトル行をキャプチャします。
	TitleRegex = regexp.MustCompile(`^#\s+(.+)`)

	// PanelRegex は "## Panel" で始まるパネル区切り行を特定します。
	PanelRegex = regexp.MustCompile(`^##\s+Panel`)

	// FieldRegex は "- key: value" 形式のフィールド行をキャプチャします。
	FieldRegex = regexp.MustCompile(`^\s*-\s*([a-zA-Z_]+):\s*(.+)`)
)
