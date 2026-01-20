package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/prompts"

	"github.com/shouni/go-gemini-client/pkg/gemini"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"github.com/shouni/go-web-exact/v2/pkg/extract"
)

const (
	// maxInputSize は読み込みを許可する最大テキストサイズ (5MB) です。
	maxInputSize = 5 * 1024 * 1024
	// maxErrorResponseLength はエラーログに含める応答抜粋の最大文字数です。
	maxErrorResponseLength = 200
)

// jsonBlockRegex は Markdown 形式の JSON ブロックを抽出するための正規表現です。
var jsonBlockRegex = regexp.MustCompile("(?s)```(?:json)?\\s*(.*\\S)\\s*```")

type MangaScriptRunner struct {
	cfg           config.Config
	extractor     *extract.Extractor
	promptBuilder prompts.ScriptPrompt
	aiClient      gemini.GenerativeModel
	reader        remoteio.InputReader
}

// NewMangaScriptRunner は依存関係を注入して初期化します。
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
func (sr *MangaScriptRunner) Run(ctx context.Context, sourceURL string, mode string) (*domain.MangaResponse, error) {
	slog.Info("ScriptRunner: 処理を開始", "url", sourceURL)

	// 1. ソースからテキストを取得
	inputText, err := sr.getTextFromSource(ctx, sourceURL)
	if err != nil {
		return nil, err
	}

	// 2. プロンプトの構築
	templateData := prompts.TemplateData{InputText: inputText}
	finalPrompt, err := sr.promptBuilder.Build(mode, templateData)
	if err != nil {
		return nil, fmt.Errorf("プロンプトの構築に失敗しました: %w", err)
	}

	// 3. Gemini API を呼び出し
	slog.Info("ScriptRunner: Gemini APIを呼び出し中", "model", sr.cfg.GeminiModel)
	resp, err := sr.aiClient.GenerateContent(ctx, sr.cfg.GeminiModel, finalPrompt)
	if err != nil {
		return nil, fmt.Errorf("Geminiによるコンテンツ生成に失敗しました: %w", err)
	}

	// 4. AI の応答をパース
	manga, err := sr.parseResponse(resp.Text)
	if err != nil {
		return nil, err
	}

	return manga, nil
}

// getTextFromSource はソースの種類を判定し、制限サイズ内でテキストを取得します。
func (sr *MangaScriptRunner) getTextFromSource(ctx context.Context, sourceURL string) (string, error) {
	if remoteio.IsGCSURI(sourceURL) {
		return sr.readFromGCS(ctx, sourceURL)
	}
	return sr.readFromWeb(ctx, sourceURL)
}

// readFromGCS は GCS からファイルを読み込みます。
func (sr *MangaScriptRunner) readFromGCS(ctx context.Context, url string) (string, error) {
	rc, err := sr.reader.Open(ctx, url)
	if err != nil {
		return "", fmt.Errorf("GCSファイルのオープンに失敗しました: %w", err)
	}
	defer rc.Close()

	limitedReader := io.LimitReader(rc, maxInputSize)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("GCSファイルの読み込みに失敗しました: %w", err)
	}

	if int64(len(content)) >= maxInputSize {
		slog.WarnContext(ctx, "GCS入力が制限サイズに達したため切り捨てられました",
			"url", url,
			"limit_bytes", maxInputSize)
	}
	return string(content), nil
}

// readFromWeb は Web サイトからテキストを抽出し、メモリ効率を考慮してサイズ制限を適用します。
func (sr *MangaScriptRunner) readFromWeb(ctx context.Context, url string) (string, error) {
	text, _, err := sr.extractor.FetchAndExtractText(ctx, url)
	if err != nil {
		return "", fmt.Errorf("URLからのテキスト抽出に失敗しました: %w", err)
	}

	// バイト単位でチェックし、効率的に安全な位置で切り捨てる
	if len(text) > maxInputSize {
		slog.WarnContext(ctx, "Web入力が制限サイズを超えたため切り捨てます",
			"url", url,
			"limit_bytes", maxInputSize)

		end := maxInputSize
		// マルチバイト文字の途中で切断されるのを防ぐため、有効なルーンの開始位置まで遡る
		for end > 0 && !utf8.RuneStart(text[end]) {
			end--
		}
		text = text[:end]
	}
	return text, nil
}

// parseResponse は AI の応答から JSON を抽出し、構造体に変換します。
func (sr *MangaScriptRunner) parseResponse(raw string) (*domain.MangaResponse, error) {
	raw = strings.TrimSpace(raw)
	var rawJSON string

	if matches := jsonBlockRegex.FindStringSubmatch(raw); len(matches) > 1 {
		rawJSON = matches[1]
	} else {
		first := strings.Index(raw, "{")
		last := strings.LastIndex(raw, "}")
		if first != -1 && last != -1 && last > first {
			rawJSON = raw[first : last+1]
		} else {
			rawJSON = raw
		}
	}

	var manga domain.MangaResponse
	if err := json.Unmarshal([]byte(rawJSON), &manga); err != nil {
		return nil, fmt.Errorf("AI応答JSONの解析に失敗しました (抜粋: %q): %w",
			truncateString(raw, maxErrorResponseLength), err)
	}

	return &manga, nil
}

// truncateString は指定された長さで文字列を安全に切り捨てます。
func truncateString(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	// 安全な切り捨てのためにルーン単位で処理
	runes := []rune(s)
	return string(runes[:maxLen]) + "..."
}
