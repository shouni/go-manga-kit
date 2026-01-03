package publisher

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/shouni/gemini-image-kit/pkg/domain"
)

// MarkdownPublisher は、生成結果を構造化された Markdown 形式で出力する役割を担います。
type MarkdownPublisher struct {
	// タグ除去用の正規表現などは、必要に応じてドメイン層やdirectorから注入される想定です。
}

func NewMarkdownPublisher() *MarkdownPublisher {
	return &MarkdownPublisher{}
}

// BuildFinalMarkdown は、漫画のタイトル、画像パス、パネル情報を統合して
// go-text-format が解釈可能な Markdown 文字列を生成します。
func (mp *MarkdownPublisher) BuildFinalMarkdown(title string, imagePaths []string, pages []domain.MangaPage) string {
	var sb strings.Builder

	// 1. タイトルの出力
	sb.WriteString(fmt.Sprintf("# %s\n\n", title))

	for i, page := range pages {
		// 画像パスの決定（画像が足りない場合はプレースホルダー）
		imagePath := "placeholder.png"
		if i < len(imagePaths) {
			imagePath = imagePaths[i]
		}

		// 2. パネルヘッダーの出力
		sb.WriteString(fmt.Sprintf("## Panel: %s\n", imagePath))
		sb.WriteString("- layout: standard\n")

		if page.Dialogue != "" {
			// 3. 話者の処理（ハッシュ化による安全なクラス名生成）
			speaker := page.SpeakerID
			if speaker == "" {
				speaker = "narration"
			}

			// 日本語名などのマルチバイト文字を CSS 安全な ID に変換
			h := sha256.New()
			h.Write([]byte(speaker))
			speakerClass := fmt.Sprintf("speaker-%s", hex.EncodeToString(h.Sum(nil))[:10])

			sb.WriteString(fmt.Sprintf("- speaker: %s\n", speakerClass))

			// 4. セリフの出力（改行コードを適切に処理）
			// text 属性は go-text-format の複数行パースに対応させる
			sb.WriteString(fmt.Sprintf("- text: %s\n", strings.TrimSpace(page.Dialogue)))

			// 5. 演出スタイルの注入（本来は director から渡されるデータを使用）
			// ここでは記憶した getDialogueStyle のデフォルト挙動を反映
			sb.WriteString(mp.getDefaultStyle(i))
		} else {
			sb.WriteString("- type: none\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// getDefaultStyle は、director が不在の場合のデフォルトの吹き出し配置を返します。
func (mp *MarkdownPublisher) getDefaultStyle(index int) string {
	if index%2 == 0 {
		return "- tail: top\n- bottom: 10%\n- left: 10%\n"
	}
	return "- tail: bottom\n- top: 10%\n- right: 10%\n"
}
