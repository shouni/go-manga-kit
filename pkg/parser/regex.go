package parser

import "regexp"

var (
	// TitleRegex は "# タイトル" をキャプチャするのだ
	TitleRegex = regexp.MustCompile(`^#\s+(.+)`)

	// PanelRegex は "## Panel" で始まる行を特定するのだ
	PanelRegex = regexp.MustCompile(`^##\s+Panel`)

	// FieldRegex は "- key: value" 形式をキャプチャするのだ
	FieldRegex = regexp.MustCompile(`^\s*-\s*([a-zA-Z_]+):\s*(.+)`)
)
