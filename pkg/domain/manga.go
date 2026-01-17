package domain

// MangaResponse は AI モデルから返される台本全体の構造です。
type MangaResponse struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Panels      []Panel `json:"panels"`
}

// Panel は漫画の1ページまたは1パネルの構成、セリフ、話者情報を保持します。
type Panel struct {
	Page         int    `json:"page"`
	VisualAnchor string `json:"visual_anchor"`
	Dialogue     string `json:"dialogue"`
	SpeakerID    string `json:"speaker_id"`
	ReferenceURL string `json:"reference_url"`

	// GeneratedImageURI は 生成された個別パネル画像の File API URI。
	GeneratedImageURI string `json:"-"`
}

// Page は物理的な1枚の画像（複数のパネルを統合したもの）を表します。
// 複数のパネルを1枚の画像に合成する場合に活用します。
type Page struct {
	PageNumber int
	ImageURL   string
	Panels     []Panel
}
