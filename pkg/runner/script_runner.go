package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/prompts"

	"github.com/shouni/go-gemini-client/pkg/gemini"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"github.com/shouni/go-web-exact/v2/pkg/extract"
)

// maxInputSize は読み込みを許可する最大テキストサイズ (5MB) です。
const maxInputSize = 5 * 1024 * 1024

// jsonBlockRegex は Markdown 形式の JSON ブロックを抽出するための正規表現です。
var jsonBlockRegex = regexp.MustCompile("(?s)```(?:json)?\\s*(.*\\S)\\s*```")

type MangaScriptRunner struct {
	cfg           config.Config
	extractor     *extract.Extractor
	promptBuilder prompts.ScriptPrompt
	aiClient      gemini.GenerativeModel
	reader        remoteio.InputReader
}

// NewMangaScriptRunner は依存関係（ビルダーを含む）を注入して初期化します。
func NewMangaScriptRunner(
	cfg config.Config,
	ext *extract.Extractor,
	pb prompts.ScriptPrompt,
	ai gemini.GenerativeModel,
	r remoteio.InputReader,
) *MangaScriptRunner {
	return &MangaScriptRunner{
		cfg:           cfg,
		extractor:     ext,
		promptBuilder: pb,
		aiClient:      ai,
		reader:        r,
	}
}

// Run は Web ページまたは GCS から内容を抽出し、Gemini を用いて漫画の台本 JSON を生成します。
func (sr *MangaScriptRunner) Run(ctx context.Context, scriptURL string, mode string) (*domain.MangaResponse, error) {
	slog.Info("ScriptRunner: テキストを抽出中", "url", scriptURL)

	var inputText string
	var err error

	// 1. ソースの種類（GCS または Web）に応じてテキストを取得
	if remoteio.IsGCSURI(scriptURL) {
		rc, err := sr.reader.Open(ctx, scriptURL)
		if err != nil {
			return nil, fmt.Errorf("GCSファイルのオープンに失敗しました: %w", err)
		}
		defer rc.Close()

		// 巨大なファイルによるメモリ不足 (OOM) を防ぐため LimitReader を使用
		limitedReader := io.LimitReader(rc, maxInputSize)
		content, err := io.ReadAll(limitedReader)
		if err != nil {
			return nil, fmt.Errorf("GCSファイルの読み込みに失敗しました: %w", err)
		}

		// 読み込んだサイズが上限に達した場合、警告ログを出力
		if len(content) >= int(maxInputSize) {
			slog.WarnContext(ctx, "GCSからの入力がサイズ制限に達したため、切り捨てられた可能性があります",
				"url", scriptURL,
				"制限サイズ", maxInputSize,
				"読み込みサイズ", len(content))
		}
		inputText = string(content)
	} else {
		// Web サイトからテキストを抽出
		inputText, err = sr.extractContent(ctx, scriptURL)
		if err != nil {
			return nil, fmt.Errorf("URLからのテキスト抽出に失敗しました: %w", err)
		}
	}

	// 2. プロンプトの構築
	// アプリ側で定義された TemplateData を使用してプロンプトを組み立てます
	templateData := prompts.TemplateData{InputText: inputText}
	finalPrompt, err := sr.promptBuilder.Build(mode, templateData)
	if err != nil {
		return nil, fmt.Errorf("プロンプトの構築に失敗しました: %w", err)
	}

	// 3. Gemini API を呼び出し
	slog.Info("ScriptRunner: Gemini APIを呼び出し中", "モデル", sr.cfg.GeminiModel)
	resp, err := sr.aiClient.GenerateContent(ctx, sr.cfg.GeminiModel, finalPrompt)
	if err != nil {
		return nil, fmt.Errorf("Geminiによるコンテンツ生成に失敗しました: %w", err)
	}

	// 4. AI の応答をパースして構造化データ (JSON) に変換
	manga, err := sr.parseResponse(resp.Text)
	if err != nil {
		return nil, err
	}

	return manga, nil
}

// extractContent は指定された URL からテキストコンテンツを抽出します。
func (sr *MangaScriptRunner) extractContent(ctx context.Context, url string) (string, error) {
	text, _, err := sr.extractor.FetchAndExtractText(ctx, url)
	if err != nil {
		return "", fmt.Errorf("URLの取得または解析に失敗しました: %w", err)
	}
	return text, nil
}

// parseResponse は AI からの生の応答から JSON 部分を取り出し、構造体に変換します。
func (sr *MangaScriptRunner) parseResponse(raw string) (*domain.MangaResponse, error) {
	raw = strings.TrimSpace(raw)
	var rawJSON string

	// Markdown の JSON ブロック (```json ... ```) を優先的に抽出
	matches := jsonBlockRegex.FindStringSubmatch(raw)
	if len(matches) > 1 {
		rawJSON = matches[1]
	} else {
		// ブロックが見つからない場合、最初と最後の波括弧を探す
		firstBracket := strings.Index(raw, "{")
		lastBracket := strings.LastIndex(raw, "}")
		if firstBracket != -1 && lastBracket != -1 && lastBracket > firstBracket {
			rawJSON = raw[firstBracket : lastBracket+1]
		} else {
			rawJSON = raw
		}
	}

	var manga domain.MangaResponse
	if err := json.Unmarshal([]byte(rawJSON), &manga); err != nil {
		// 解析失敗時は、原因特定のために応答の冒頭を含めてエラーを返します
		return nil, fmt.Errorf("AIからの応答 JSON の解析に失敗しました (応答抜粋: %q): %w", truncateString(raw, 200), err)
	}

	return &manga, nil
}

// truncateString は文字列を指定された最大長で安全に切り捨てます。
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
