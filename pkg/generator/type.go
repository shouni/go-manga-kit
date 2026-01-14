package generator

import (
	"github.com/shouni/go-manga-kit/pkg/domain"

	"github.com/shouni/gemini-image-kit/pkg/generator"
)

const (
	// MaxPanelsPerPage は1枚の漫画ページに含めるパネルの最大数です。
	MaxPanelsPerPage = 6

	// PanelAspectRatio は単体パネル（1コマ）の推奨アスペクト比です。
	PanelAspectRatio = "16:9"

	// PageAspectRatio は統合ページ全体の推奨アスペクト比です。
	PageAspectRatio = "3:4"
)

type MangaGenerator struct {
	ImgGen     generator.ImageGenerator
	Characters map[string]domain.Character
}
