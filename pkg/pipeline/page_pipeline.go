package pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/domain"
)

// PagePipeline は複数のパネルを1枚の漫画ページとして統合生成する汎用部品なのだ。
type PagePipeline struct {
	manga       Pipeline
	styleSuffix string
}

func NewPagePipeline(manga Pipeline, styleSuffix string) *PagePipeline {
	return &PagePipeline{
		manga:       manga,
		styleSuffix: styleSuffix}
}

// ExecuteMangaPage は構造化された台本を基に、1枚の統合漫画画像を生成する
func (pp *PagePipeline) ExecuteMangaPage(ctx context.Context, manga domain.MangaResponse) (*imagedom.ImageResponse, error) {
	//markdownParser := parser.NewParser(scriptURL)
	//manga, err := markdownParser.Parse(markdownContent)
	//if err != nil {
	//	return nil, fmt.Errorf("Markdownのパースに失敗しました: %w", err)
	//}

	// 1. 参照URLの収集
	refURLs := pp.collectReferences(manga.Pages, pp.manga.Characters)

	// 2. 巨大な統合プロンプトの構築
	fullPrompt := pp.buildUnifiedPrompt(manga, pp.manga.Characters, refURLs)

	// 3. シード値の決定（最初のパネルのキャラを優先）
	var defaultSeed *int64
	if len(manga.Pages) > 0 {
		char := pp.findCharacter(manga.Pages[0].SpeakerID, pp.manga.Characters)
		if char != nil && char.Seed > 0 {
			s := char.Seed
			defaultSeed = &s
		}
	}

	req := imagedom.ImagePageRequest{
		Prompt:         fullPrompt,
		NegativePrompt: "deformed faces, mismatched eyes, cross-eyed, low-quality faces, blurry facial features, melting faces, extra limbs, merged panels, messy lineart, distorted anatomy",
		AspectRatio:    "3:4",
		Seed:           defaultSeed,
		ReferenceURLs:  refURLs,
	}

	return pp.manga.ImgGen.GenerateMangaPage(ctx, req)
}

// findCharacter は SpeakerID（名前またはハッシュ化ID）からキャラを特定するのだ
func (pp *PagePipeline) findCharacter(speakerID string, characters map[string]domain.Character) *domain.Character {
	sid := strings.ToLower(speakerID)
	h := sha256.New()
	for _, char := range characters {
		h.Reset()
		h.Write([]byte(char.ID))
		hash := hex.EncodeToString(h.Sum(nil))
		if sid == "speaker-"+hash[:10] {
			return &char
		}
	}
	cleanID := strings.TrimPrefix(sid, "speaker-")
	if char, ok := characters[cleanID]; ok {
		return &char
	}
	return nil
}

// buildUnifiedPrompt は AIに対してマンガのレイアウトとDNAを叩き込むのだ！
func (pp *PagePipeline) buildUnifiedPrompt(manga domain.MangaResponse, characters map[string]domain.Character, refURLs []string) string {
	var sb strings.Builder
	urlToIndex := make(map[string]int)
	for i, url := range refURLs {
		urlToIndex[url] = i + 1
	}

	// 1. レイアウト定義
	sb.WriteString("### MANDATORY FORMAT: MULTI-PANEL MANGA PAGE COMPOSITION ###\n")
	sb.WriteString(fmt.Sprintf("- TOTAL PANELS: This page MUST contain exactly %d distinct panels.\n", len(manga.Pages)))
	sb.WriteString("- STRUCTURE: A professional Japanese manga spread with clear frame borders.\n")
	sb.WriteString("- READING ORDER: Right-to-Left, Top-to-Bottom.\n")
	sb.WriteString("- GUTTERS: Ultra-thin, crisp hairline dividers. NO OVERLAPPING.\n\n")

	// 2. グローバルスタイル
	sb.WriteString("### GLOBAL VISUAL STYLE ###\n")
	if pp.styleSuffix != "" {
		sb.WriteString(fmt.Sprintf("- STYLE_DNA: %s\n", pp.styleSuffix))
	}
	sb.WriteString("- RENDERING: Sharp clean lineart, vibrant colors, cinematic lighting.\n\n")

	// 3. キャラクターDNA
	sb.WriteString("### CHARACTER DNA (MASTER IDENTITY) ###\n")
	for _, char := range characters {
		if idx, found := urlToIndex[char.ReferenceURL]; found {
			cues := strings.Join(char.VisualCues, ", ")
			sb.WriteString(fmt.Sprintf("- [%s]: IDENTITY_REF_#%d. FEATURES: %s\n", char.Name, idx, cues))
		}
	}
	sb.WriteString("\n")

	// 4. パネル詳細
	for i, page := range manga.Pages {
		panelNum := i + 1
		sb.WriteString("===========================================\n")
		sb.WriteString(fmt.Sprintf("### [INDEPENDENT PANEL %d OF %d] ###\n", panelNum, len(manga.Pages)))

		if i == 0 || strings.Contains(page.VisualAnchor, "大ゴマ") {
			sb.WriteString("- SIZE: PRIMARY FEATURE PANEL. Large and impactful.\n")
		}

		char := pp.findCharacter(page.SpeakerID, characters)
		if char != nil {
			if idx, found := urlToIndex[char.ReferenceURL]; found {
				sb.WriteString(fmt.Sprintf("- SUBJECT: %s (DNA_REF_#%d). VISUALS: %s\n", char.Name, idx, strings.Join(char.VisualCues, ", ")))
			}
		}

		if idx, found := urlToIndex[page.ReferenceURL]; found {
			sb.WriteString(fmt.Sprintf("- COMPOSITION: Refer to IMAGE_REF_#%d.\n", idx))
		}

		sb.WriteString(fmt.Sprintf("- SCENE_ACTION: %s\n", page.VisualAnchor))
		if page.Dialogue != "" {
			sb.WriteString(fmt.Sprintf("- TEXT_BUBBLE: \"%s\"\n", page.Dialogue))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// collectReferences は必要な全ての画像URLを重複なく収集するのだ
func (pp *PagePipeline) collectReferences(pages []domain.MangaPage, characters map[string]domain.Character) []string {
	urlMap := make(map[string]struct{})
	var urls []string
	for _, p := range pages {
		if char := pp.findCharacter(p.SpeakerID, characters); char != nil && char.ReferenceURL != "" {
			if _, exists := urlMap[char.ReferenceURL]; !exists {
				urlMap[char.ReferenceURL] = struct{}{}
				urls = append(urls, char.ReferenceURL)
			}
		}
	}
	for _, p := range pages {
		if p.ReferenceURL != "" {
			if _, exists := urlMap[p.ReferenceURL]; !exists {
				urlMap[p.ReferenceURL] = struct{}{}
				urls = append(urls, p.ReferenceURL)
			}
		}
	}
	return urls
}
