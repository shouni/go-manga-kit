package generator

import (
	"github.com/shouni/go-manga-kit/pkg/domain"

	"github.com/shouni/gemini-image-kit/pkg/generator"
)

type MangaGenerator struct {
	ImgGen     generator.ImageGenerator
	Characters map[string]domain.Character
}
