package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
)

var (
	titleRegex = regexp.MustCompile(`^#\s+(.+)`)
	panelRegex = regexp.MustCompile(`^##\s+Panel:?`)
	fieldRegex = regexp.MustCompile(`^\s*-\s*([a-zA-Z_]+):\s*(.+)`)
)

// MangaPageAdapter は画像生成エンジン（Nano Banana等）へのインターフェースなのだ
type MangaPageAdapter interface {
	GenerateMangaPage(ctx context.Context, req imagedom.ImagePageRequest) (*imagedom.ImageResponse, error)
}

// MangaPageRunner は複数のコマを1枚の漫画ページとして統合生成するコアロジックを担うのだ
type MangaPageRunner struct {
	adapter     MangaPageAdapter
	characters  map[string]*domain.Character
	styleSuffix string // プロンプトの末尾に強制注入するスタイル指定なのだ
	baseURL     string // GCS等のアセット参照用ベースURLなのだ
}

// NewMangaPageRunner は MangaPageRunner を初期化し、GCSスキームのURL変換も行うのだ
func NewMangaPageRunner(adapter MangaPageAdapter, characters map[string]*domain.Character, styleSuffix string, scriptURL string) *MangaPageRunner {
	baseURL := ""
	u, err := url.Parse(scriptURL)
	// GCSのgs://スキームをブラウザやAIが解釈可能なhttps形式に変換して公開URLのベースを構築するのだ
	if err == nil && u.Scheme == "gs" {
		// path.Dirでファイル名を除いたディレクトリ部分を取得するのだ
		dirPath := path.Dir(u.Path)
		baseURL = fmt.Sprintf("https://storage.googleapis.com/%s%s/", u.Host, dirPath)
	}

	return &MangaPageRunner{
		adapter:     adapter,
		characters:  characters,
		styleSuffix: styleSuffix,
		baseURL:     baseURL,
	}
}

// findCharacter は SpeakerID（名前またはハッシュ化ID）から対応するキャラクター情報を検索するのだ
func (r *MangaPageRunner) findCharacter(speakerID string) *domain.Character {
	sid := strings.ToLower(speakerID)
	// SHA1ハッシュで生成されたID（speaker-hash[:10]）との照合を試みるのだ
	h := sha256.New()
	for _, char := range r.characters {
		h.Reset()
		h.Write([]byte(char.ID))
		hash := hex.EncodeToString(h.Sum(nil))
		expectedID := "speaker-" + hash[:10]
		if sid == expectedID {
			return char
		}
	}
	// ハッシュでない直接のID指定（"zundamon"等）でも検索できるようにフォールバックするのだ
	cleanID := strings.TrimPrefix(sid, "speaker-")
	if char, ok := r.characters[cleanID]; ok {
		return char
	}
	return nil
}

// Run は構造化された MangaResponse をもとに、1枚の統合漫画画像を生成するのだ
func (r *MangaPageRunner) Run(ctx context.Context, manga domain.MangaResponse) (*imagedom.ImageResponse, error) {
	// 登場するキャラクターや構図のリファレンスURLを収集するのだ
	refURLs := r.collectCharacterReferences(manga.Pages)
	// 8コマグリッドやDNA保持指示を含む、巨大な統合プロンプトを構築するのだ
	fullPrompt := r.buildUnifiedPrompt(manga, refURLs)

	var defaultSeed *int64
	if len(manga.Pages) > 0 {
		// 最初のコマのキャラクターが持つ固定Seedがあれば、それをページ全体のベースにするのだ
		char := r.findCharacter(manga.Pages[0].SpeakerID)
		if char != nil && char.Seed > 0 {
			s := char.Seed
			defaultSeed = &s
		}
	}

	// 「顔の崩れ」や「コマの融合」を防ぐための強力なネガティブプロンプトなのだ！
	negativePrompt := "deformed faces, mismatched eyes, cross-eyed, low-quality faces, blurry facial features, melting faces, extra limbs, merged panels, messy lineart, distorted anatomy"

	req := imagedom.ImagePageRequest{
		Prompt:         fullPrompt,
		NegativePrompt: negativePrompt,
		AspectRatio:    "3:4", // 縦長の漫画1ページ形式なのだ
		Seed:           defaultSeed,
		ReferenceURLs:  refURLs,
	}
	return r.adapter.GenerateMangaPage(ctx, req)
}

// RunMarkdown は Markdown形式の台本をパースしてから漫画生成を実行するのだ
func (r *MangaPageRunner) RunMarkdown(ctx context.Context, markdownContent string) (*imagedom.ImageResponse, error) {
	manga, err := r.ParseMarkdown(markdownContent)
	if err != nil {
		return nil, fmt.Errorf("Markdownのパースに失敗しました: %w", err)
	}
	return r.Run(ctx, *manga)
}

// ParseMarkdown は Markdownテキストを行解析し、MangaResponse 構造体に変換するのだ
func (r *MangaPageRunner) ParseMarkdown(input string) (*domain.MangaResponse, error) {
	manga := &domain.MangaResponse{}
	lines := strings.Split(input, "\n")
	var currentPage *domain.MangaPage

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}
		// タイトル行（# Title）の抽出
		if m := titleRegex.FindStringSubmatch(trimmedLine); m != nil {
			manga.Title = strings.TrimSpace(m[1])
			continue
		}
		// パネル区切り（## Panel）の抽出と、参照URLのパス補完
		if panelRegex.MatchString(trimmedLine) {
			if currentPage != nil {
				manga.Pages = append(manga.Pages, *currentPage)
			}
			refPath := ""
			if strings.Contains(trimmedLine, ":") {
				if _, after, found := strings.Cut(trimmedLine, ":"); found {
					refPath = strings.TrimSpace(after)
				}
			}
			fullPath := refPath
			// 相対パスの場合は、baseURLを付与してフルURLにするのだ
			if refPath != "" && !strings.HasPrefix(refPath, "http") {
				fullPath = r.baseURL + refPath
			}
			currentPage = &domain.MangaPage{
				Page:         len(manga.Pages) + 1,
				ReferenceURL: fullPath,
			}
			continue
		}
		// フィールド行（- key: value）の抽出
		if currentPage != nil {
			if m := fieldRegex.FindStringSubmatch(trimmedLine); m != nil {
				key, val := strings.ToLower(m[1]), strings.TrimSpace(m[2])
				switch key {
				case "speaker":
					currentPage.SpeakerID = strings.ToLower(val)
				case "text":
					currentPage.Dialogue = val
				case "action":
					currentPage.VisualAnchor = val
				}
			}
		}
	}
	if currentPage != nil {
		manga.Pages = append(manga.Pages, *currentPage)
	}
	return manga, nil
}

// buildUnifiedPrompt は AIに対して「漫画としてのレイアウト」と「各コマのDNA」を指示する超重要プロンプトを作るのだ
func (r *MangaPageRunner) buildUnifiedPrompt(manga domain.MangaResponse, refURLs []string) string {
	var sb strings.Builder
	urlToIndex := make(map[string]int)
	for i, url := range refURLs {
		urlToIndex[url] = i + 1
	}

	// 1. 【レイアウト定義】日本式の読み順やコマ割りのルールを叩き込むのだ
	sb.WriteString("### MANDATORY FORMAT: MULTI-PANEL MANGA PAGE COMPOSITION ###\n")
	sb.WriteString("- TOTAL PANELS: This page MUST contain exactly " + fmt.Sprint(len(manga.Pages)) + " distinct panels.\n")
	sb.WriteString("- STRUCTURE: A professional Japanese manga spread with clear frame borders.\n")
	sb.WriteString("- READING ORDER: Right-to-Left, Top-to-Bottom.\n")
	sb.WriteString("- GUTTERS: Ultra-thin, crisp hairline dividers. NO OVERLAPPING. Each panel is a separate scene.\n")
	sb.WriteString("- STYLE: High-quality ink lineart, anime style, flat colors, cinematic manga lighting.\n\n")

	// 2. 【グローバルスタイル】全体の画風を一貫させるのだ
	sb.WriteString("### GLOBAL VISUAL STYLE ###\n")
	if r.styleSuffix != "" {
		sb.WriteString(fmt.Sprintf("- STYLE_DNA: %s\n", r.styleSuffix))
	}
	sb.WriteString("- RENDERING: Sharp clean lineart, vibrant colors, no blurring, high contrast.\n\n")

	// 3. 【キャラクターDNA】マスターリファレンスとして全登場人物の特徴を定義するのだ
	sb.WriteString("### CHARACTER DNA (MASTER IDENTITY) ###\n")
	for _, char := range r.characters {
		if idx, found := urlToIndex[char.ReferenceURL]; found {
			cues := strings.Join(char.VisualCues, ", ")
			sb.WriteString(fmt.Sprintf("- [%s]: IDENTITY_REF_#%d. FEATURES: %s\n", char.Name, idx, cues))
		}
	}
	sb.WriteString("\n")

	// 4. 【パネル詳細】各コマの内容を独立させて記述し、DNAの引き継ぎを徹底させるのだ
	totalPanels := len(manga.Pages)
	for i, page := range manga.Pages {
		panelNum := i + 1
		sb.WriteString(fmt.Sprintf("===========================================\n"))
		sb.WriteString(fmt.Sprintf("### [INDEPENDENT PANEL %d OF %d] ###\n", panelNum, totalPanels))

		// 最初のコマや特定の指示がある場合は「大ゴマ」として扱うのだ
		isBigPanel := i == 0 || strings.Contains(page.VisualAnchor, "大ゴマ")
		if isBigPanel {
			sb.WriteString("- SIZE: PRIMARY FEATURE PANEL. Large and impactful.\n")
		} else {
			sb.WriteString("- SIZE: COMPACT SUPPORTING PANEL. Integrated into the flow.\n")
		}

		// 右から左へ進む日本式シーケンス内での位置を示すのだ
		sb.WriteString(fmt.Sprintf("- PLACEMENT: Part of a Right-to-Left sequence, Step %d.\n", panelNum))

		// コマごとのキャラクター外見固定（DNAリファレンス）
		char := r.findCharacter(page.SpeakerID)
		if char != nil {
			if idx, found := urlToIndex[char.ReferenceURL]; found {
				cues := strings.Join(char.VisualCues, ", ")
				sb.WriteString(fmt.Sprintf("- SUBJECT: %s (DNA_REF_#%d). VISUALS: %s\n", char.Name, idx, cues))
				sb.WriteString("- SUBJECT_FIDELITY: Strict adherence to Character DNA. Consistent face and hair.\n")
			}
		}

		// Image-to-Image用の構図リファレンス指定
		if idx, found := urlToIndex[page.ReferenceURL]; found {
			sb.WriteString(fmt.Sprintf("- COMPOSITION: Refer to IMAGE_REF_#%d for camera angle and layout.\n", idx))
		}

		// シーンのアクションとセリフ（吹き出し）の指示
		sb.WriteString(fmt.Sprintf("- SCENE_ACTION: %s\n", page.VisualAnchor))
		if page.Dialogue != "" {
			sb.WriteString(fmt.Sprintf("- TEXT_BUBBLE: \"%s\" (Render speech bubble within this panel only).\n", page.Dialogue))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// collectCharacterReferences は生成に必要な全ての画像URLを重複なく収集し、AIに渡すリストを作るのだ
func (r *MangaPageRunner) collectCharacterReferences(pages []domain.MangaPage) []string {
	urlMap := make(map[string]struct{})
	var urls []string
	// まずは登場キャラクターのマスター画像を収集
	for _, p := range pages {
		if char := r.findCharacter(p.SpeakerID); char != nil && char.ReferenceURL != "" {
			if _, exists := urlMap[char.ReferenceURL]; !exists {
				urlMap[char.ReferenceURL] = struct{}{}
				urls = append(urls, char.ReferenceURL)
			}
		}
	}
	// 次に、各コマ固有の参照画像（背景や構図指示用）を収集
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
