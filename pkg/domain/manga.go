package domain

type MangaResponse struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Pages       []MangaPage `json:"pages"`
}

// MangaPage は漫画の1ページまたは1パネルの構成、セリフ、話者情報を保持します。
type MangaPage struct {
	Page         int    `json:"page"`
	VisualAnchor string `json:"visual_anchor"`
	Dialogue     string `json:"dialogue"`
	SpeakerID    string `json:"speaker_id"`
	ReferenceURL string `json:"reference_url"`
}

// Panel は最終的な画像とテキストの統合成果物です。
type Panel struct {
	PageNumber int
	PanelIndex int
	Prompt     string
	Dialogue   string
	Character  *Character
	ImageBytes []byte
}
