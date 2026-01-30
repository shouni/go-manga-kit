package generator

const (
	// MaxPanelsPerPage は1枚の漫画ページに含めるパネルの最大数です。
	MaxPanelsPerPage = 5
	// PanelAspectRatio は単体パネル（1コマ）の推奨アスペクト比です。
	PanelAspectRatio = "16:9"
	// PageAspectRatio は統合ページ全体の推奨アスペクト比です。
	PageAspectRatio = "3:4"

	// ImageSize1K は標準的な解像度の設定（1024x1024相当）です。
	ImageSize1K = "1K"
	// ImageSize2K は高解像度の設定（2048x2048相当）です。
	ImageSize2K = "2K"
	// ImageSize4K は超高解像度の設定（4096x4096相当）です。
	ImageSize4K = "4K"
)
