package generator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"strings"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/prompt"
)

// PagePipeline は複数のパネルを1枚の漫画ページとして統合生成する汎用部品なのだ。
type PagePipeline struct {
	mangaGenerator MangaGenerator
	styleSuffix    string
}

func NewPagePipeline(mangaGenerator MangaGenerator, styleSuffix string) *PagePipeline {
	return &PagePipeline{
		mangaGenerator: mangaGenerator,
		styleSuffix:    styleSuffix,
	}
}

// ExecuteMangaPage は構造化された台本を基に、1枚の統合漫画画像を生成する
func (pp *PagePipeline) ExecuteMangaPage(ctx context.Context, manga domain.MangaResponse) (*imagedom.ImageResponse, error) {
	// 共通のスタイルサフィックス（anime styleなど）を注入して生成するのだ
	pb := prompt.NewPromptBuilder(pp.mangaGenerator.Characters, pp.styleSuffix)

	// 1. 参照URLの収集
	refURLs := pp.collectReferences(manga.Pages, pp.mangaGenerator.Characters)

	// 2. 巨大な統合プロンプトの構築
	fullPrompt := pb.BuildFullPagePrompt(manga.Title, manga.Pages, refURLs)

	// TODO::プロンプトがうまくいったら削除
	//// 2. 巨大な統合プロンプトの構築
	//fullPrompt := pp.buildUnifiedPrompt(manga, pp.mangaPipeline.Characters, refURLs)

	// 3. シード値の決定（最初のパネルのキャラを優先）
	var defaultSeed *int64
	if len(manga.Pages) > 0 {
		char := pp.findCharacter(manga.Pages[0].SpeakerID, pp.mangaGenerator.Characters)
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

	return pp.mangaGenerator.ImgGen.GenerateMangaPage(ctx, req)
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

// TODO:あとで削除
// buildUnifiedPrompt は AIに対してマンガのレイアウトとDNAを設定
func (pp *PagePipeline) buildUnifiedPrompt(manga domain.MangaResponse, characters domain.CharactersMap, refURLs []string) string {
	var sb strings.Builder

	// Reference URLの逆引きマップ（どのURLが何番目のパネルか）
	urlToIndex := make(map[string]int)
	for i, url := range refURLs {
		urlToIndex[url] = i + 1
	}

	// 1. マンガ基本構造とページ構成の注入
	sb.WriteString(prompt.MangaStructureHeader)
	sb.WriteString(fmt.Sprintf("\n- TOTAL PANELS: This page MUST contain exactly %d distinct panels.\n", len(manga.Pages)))
	sb.WriteString("- NOTE: Maintain logical flow between panels while keeping frame borders clean.\n\n")

	// 2. グローバルな描画スタイル（Style DNA）
	sb.WriteString(prompt.RenderingStyle)
	if pp.styleSuffix != "" {
		// 共通の画風プロンプト（アニメ調、90年代風など）を最優先で注入
		sb.WriteString(fmt.Sprintf("- STYLE_DNA: %s\n", pp.styleSuffix))
	}
	sb.WriteString("\n")

	// 3. キャラクターDNA情報の注入 (ドメインで定義したCharactersMapをそのまま渡す)
	// BuildCharacterIdentitySection 内で名前とVisualCuesが紐付けられるのだ
	sb.WriteString(prompt.BuildCharacterIdentitySection(characters))

	// TODO::どのパネルを大きくするか、ひとまずランダム値に
	numPanels := len(manga.Pages)
	bigPanelIndex := -1 // デフォルトは「大きいパネルなし」
	if numPanels > 0 {
		bigPanelIndex = rand.IntN(numPanels)
	}

	// 4. 各パネル（コマ）の個別指示ループ
	for i, page := range manga.Pages {
		panelNum := i + 1

		// 最初のコマをメイン（BigPanel）にする演出ロジック
		isBig := i == bigPanelIndex
		sb.WriteString(prompt.BuildPanelHeader(panelNum, len(manga.Pages), isBig))

		// 参照画像のインデックス紐付け
		// もしこのパネルに対応するImage-to-Image用のURLがあれば、AIにその番号を意識させるのだ
		if i < len(refURLs) {
			sb.WriteString(fmt.Sprintf("- REFERENCE: See input_file_%d for character posing and composition reference.\n", panelNum))
		}

		// 実際のシーン描写（VisualAnchor）
		// ここに「めたんが餅を食べている」といった具体的な指示が入る
		sb.WriteString(fmt.Sprintf("- ACTION/SCENE: %s\n", page.VisualAnchor))

		// セリフ情報
		// 台詞自体は画像に直接描画されなくても、AIが「誰が何を喋っている状況か」を理解するのに不可欠なのだ
		if page.Dialogue != "" {
			sb.WriteString(fmt.Sprintf("- DIALOGUE_CONTEXT: [%s] says \"%s\"\n", page.SpeakerID, page.Dialogue))
		}
	}

	// 最後に一貫性を念押しするのだ
	sb.WriteString("\n### FINAL REMINDER: Ensure character identity consistency using Character Master Definitions. ###")

	return sb.String()
}

// TODO:プロンプトの調整用
//// buildUnifiedPrompt は AIに対してマンガのレイアウトとDNAを叩き込むのだ！
//func (pp *PagePipeline) buildUnifiedPrompt(manga domain.MangaResponse, characters map[string]domain.Character, refURLs []string) string {
//	var sb strings.Builder
//	urlToIndex := make(map[string]int)
//	for i, url := range refURLs {
//		urlToIndex[url] = i + 1
//	}
//
//	// 1. レイアウト定義
//	sb.WriteString("### MANDATORY FORMAT: MULTI-PANEL MANGA PAGE COMPOSITION ###\n")
//	sb.WriteString(fmt.Sprintf("- TOTAL PANELS: This page MUST contain exactly %d distinct panels.\n", len(manga.Pages)))
//	sb.WriteString("- STRUCTURE: A professional Japanese manga spread with clear frame borders.\n")
//	sb.WriteString("- READING ORDER: Right-to-Left, Top-to-Bottom.\n")
//	sb.WriteString("- GUTTERS: Ultra-thin, crisp hairline dividers. NO OVERLAPPING.\n\n")
//
//	// 2. グローバルスタイル
//	sb.WriteString("### GLOBAL VISUAL STYLE ###\n")
//	if pp.styleSuffix != "" {
//		sb.WriteString(fmt.Sprintf("- STYLE_DNA: %s\n", pp.styleSuffix))
//	}
//	sb.WriteString("- RENDERING: Sharp clean lineart, vibrant colors, cinematic lighting.\n\n")
//
//	// 3. キャラクターDNA
//	sb.WriteString("### CHARACTER DNA (MASTER IDENTITY) ###\n")
//	for _, char := range characters {
//		if idx, found := urlToIndex[char.ReferenceURL]; found {
//			cues := strings.Join(char.VisualCues, ", ")
//			sb.WriteString(fmt.Sprintf("- [%s]: IDENTITY_REF_#%d. FEATURES: %s\n", char.Name, idx, cues))
//		}
//	}
//	sb.WriteString("\n")
//
//	// 4. パネル詳細
//	for i, page := range manga.Pages {
//		panelNum := i + 1
//		sb.WriteString("===========================================\n")
//		sb.WriteString(fmt.Sprintf("### [INDEPENDENT PANEL %d OF %d] ###\n", panelNum, len(manga.Pages)))
//
//		if i == 0 || strings.Contains(page.VisualAnchor, "大ゴマ") {
//			sb.WriteString("- SIZE: PRIMARY FEATURE PANEL. Large and impactful.\n")
//		}
//
//		char := pp.findCharacter(page.SpeakerID, characters)
//		if char != nil {
//			if idx, found := urlToIndex[char.ReferenceURL]; found {
//				sb.WriteString(fmt.Sprintf("- SUBJECT: %s (DNA_REF_#%d). VISUALS: %s\n", char.Name, idx, strings.Join(char.VisualCues, ", ")))
//			}
//		}
//
//		if idx, found := urlToIndex[page.ReferenceURL]; found {
//			sb.WriteString(fmt.Sprintf("- COMPOSITION: Refer to IMAGE_REF_#%d.\n", idx))
//		}
//
//		sb.WriteString(fmt.Sprintf("- SCENE_ACTION: %s\n", page.VisualAnchor))
//		if page.Dialogue != "" {
//			sb.WriteString(fmt.Sprintf("- TEXT_BUBBLE: \"%s\"\n", page.Dialogue))
//		}
//		sb.WriteString("\n")
//	}
//
//	return sb.String()
//}

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
