package prompt

import (
	"fmt"
	"math/rand/v2"
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

	// 1. マンガ全体構造の定義
	sb.WriteString(MangaStructureHeader)
	sb.WriteString(fmt.Sprintf("\n- TOTAL PANELS: This page MUST contain exactly %d distinct panels.\n", len(pages)))

	// 2. タイトルと共通スタイルの適用
	// Note: 漫画タイトルは個別の画像生成プロンプトには不要なため、この処理は無効化しています。
	//	sb.WriteString(fmt.Sprintf("\n### TITLE: %s ###\n", mangaTitle))
	sb.WriteString(RenderingStyle)
	if pb.defaultSuffix != "" {
		sb.WriteString(fmt.Sprintf("- STYLE_DNA: %s\n", pb.defaultSuffix))
	}
	sb.WriteString("\n")

	// 3. 登場キャラクターの定義セクション
	sb.WriteString(BuildCharacterIdentitySection(pb.characterMap))

	// 4. ランダムな大ゴマの決定
	// ページの演出に多様性を持たせるため、ランダムに1つのパネルを大ゴマとして指定する。
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
func (pb *PromptBuilder) BuildUnifiedPrompt(page domain.MangaPage, speakerID string) (string, string, int64) {
	// 1. キャラクター設定の注入
	var visualParts []string
	var targetSeed int64

	if char, ok := pb.characterMap[speakerID]; ok {
		// 登録済みキャラなら、そのDNA（VisualCuesとSeed）を完全に継承するのだ！
		if len(char.VisualCues) > 0 {
			visualParts = append(visualParts, char.VisualCues...)
		}
		targetSeed = char.Seed
	} else {
		// 登録がない場合は、名前から「いつもの値」を決定論的に生成するのだ。
		// ※ speakerIDが空でも GetSeedFromName がよしなにやってくれる想定なのだ。
		targetSeed = domain.GetSeedFromName(speakerID, pb.characterMap)

		// 登録がないキャラでも、名前くらいはヒントとして入れておくとAIが助かるのだ。
		if speakerID != "" {
			visualParts = append(visualParts, speakerID)
		}
	}

	// 2. アクション/ビジュアルアンカーの追加
	if page.VisualAnchor != "" {
		visualParts = append(visualParts, page.VisualAnchor)
	}

	visualParts = append(visualParts, CinematicTags)

	// 3. デフォルトサフィックスの結合
	if pb.defaultSuffix != "" {
		visualParts = append(visualParts, pb.defaultSuffix)
	}

	// 4. カンマ区切りでクリーンに結合
	var cleanParts []string
	for _, p := range visualParts {
		if s := strings.TrimSpace(p); s != "" {
			cleanParts = append(cleanParts, s)
		}
	}

	return strings.Join(cleanParts, ", "), MangaNegativePrompt, targetSeed
}
