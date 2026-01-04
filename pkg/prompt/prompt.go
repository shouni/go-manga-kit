package prompt

import (
	"fmt"
	"math/rand/v2" // 最新の Go 1.22+ 推奨なのだ
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"
)

// PromptBuilder は、キャラクター情報を考慮してAIプロンプトを構築します。
type PromptBuilder struct {
	characterMap  domain.CharactersMap
	defaultSuffix string // "anime style, high quality" 等の共通サフィックス
}

// NewPromptBuilder は新しい PromptBuilder を生成します。
func NewPromptBuilder(chars domain.CharactersMap, suffix string) *PromptBuilder {
	return &PromptBuilder{
		characterMap:  chars,
		defaultSuffix: suffix,
	}
}

// BuildFullPagePrompt は、全パネル情報と参照URLを統合してプロンプトを構築します。
func (pb *PromptBuilder) BuildFullPagePrompt(mangaTitle string, pages []domain.MangaPage, refURLs []string) string {
	var sb strings.Builder

	// 0. Reference URLの逆引きマップ（どのURLが何番目か）を準備
	urlToIndex := make(map[string]int)
	for i, url := range refURLs {
		urlToIndex[url] = i + 1
	}

	// 1. マンガ全体構造の定義
	sb.WriteString(MangaStructureHeader)
	sb.WriteString(fmt.Sprintf("\n- TOTAL PANELS: This page MUST contain exactly %d distinct panels.\n", len(pages)))

	// 2. タイトルと共通スタイルの適用
	sb.WriteString(fmt.Sprintf("\n### TITLE: %s ###\n", mangaTitle))
	sb.WriteString(RenderingStyle)
	if pb.defaultSuffix != "" {
		sb.WriteString(fmt.Sprintf("- STYLE_DNA: %s\n", pb.defaultSuffix))
	}
	sb.WriteString("\n")

	// 3. 登場キャラクターの定義セクション
	sb.WriteString(BuildCharacterIdentitySection(pb.characterMap))

	// 4. ランダムな大ゴマの決定
	numPanels := len(pages)
	bigPanelIndex := -1
	if numPanels > 0 {
		bigPanelIndex = rand.IntN(numPanels)
	}

	// 5. 各パネルの指示
	for i, page := range pages {
		panelNum := i + 1
		isBig := (i == bigPanelIndex)

		sb.WriteString(BuildPanelHeader(panelNum, numPanels, isBig))
		// もし refURLs が渡されており、このパネルに対応する参照画像がある場合
		if i < len(refURLs) {
			sb.WriteString(fmt.Sprintf("- REFERENCE: Use input_file_%d for visual guidance on posing and layout.\n", panelNum))
		}

		sb.WriteString(fmt.Sprintf("- ACTION/SCENE: %s\n", page.VisualAnchor))
		if page.Dialogue != "" {
			sb.WriteString(fmt.Sprintf("- DIALOGUE_CONTEXT: [%s] says \"%s\"\n", page.SpeakerID, page.Dialogue))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// BuildUnifiedPrompt は、単体パネル用のプロンプトとシード値を生成します。
func (pb *PromptBuilder) BuildUnifiedPrompt(page domain.MangaPage, speakerID string) (string, int64) {
	var parts []string
	var seed int64

	// 1. キャラクター設定の注入
	if char, ok := pb.characterMap[speakerID]; ok {
		if len(char.VisualCues) > 0 {
			parts = append(parts, char.VisualCues...)
		}
		seed = char.Seed
	} else {
		// 登録がない、またはSpeakerIDが空の場合は名前からシードを生成するのだ
		// domainに定義した GetSeedFromName を使うのが正解なのだ
		seed = domain.GetSeedFromName(speakerID, pb.characterMap)
	}

	// 2. アクション/ビジュアルアンカーの追加
	if page.VisualAnchor != "" {
		parts = append(parts, page.VisualAnchor)
	}

	// 3. デフォルトサフィックスの結合
	if pb.defaultSuffix != "" {
		parts = append(parts, pb.defaultSuffix)
	}

	// 4. カンマ区切りでクリーンに結合
	var cleanParts []string
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			cleanParts = append(cleanParts, s)
		}
	}

	return strings.Join(cleanParts, ", "), seed
}
