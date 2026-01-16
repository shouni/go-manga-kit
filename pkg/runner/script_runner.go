package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/prompts"

	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"github.com/shouni/go-web-exact/v2/pkg/extract"
)

var jsonBlockRegex = regexp.MustCompile("(?s)```(?:json)?\\s*(.*\\S)\\s*```")

type MangaScriptRunner struct {
	cfg           config.Config
	extractor     *extract.Extractor
	promptBuilder prompts.PromptBuilder
	aiClient      gemini.GenerativeModel
	reader        remoteio.InputReader
}

// NewMangaScriptRunner は依存関係（ビルダーを含む）を注入して初期化します。
func NewMangaScriptRunner(
	cfg config.Config,
	ext *extract.Extractor,
	pb prompts.PromptBuilder,
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

// Run は Web ページの内容を抽出し、Gemini を用いて漫画の台本 JSON を生成します。
func (sr *MangaScriptRunner) Run(ctx context.Context, scriptURL string, mode string) (*domain.MangaResponse, error) {
	slog.Info("ScriptRunner: Extracting text", "url", scriptURL)

	// 1. Web サイトからテキストを抽出
	inputText, err := sr.extractContent(ctx, scriptURL)
	if err != nil {
		return nil, err
	}

	// TemplateData 構造体を使用して InputText を流し込みます
	templateData := prompts.TemplateData{InputText: inputText}
	finalPrompt, promptErr := sr.promptBuilder.Build(mode, templateData)
	if promptErr != nil {
		err = fmt.Errorf("プロンプト生成に失敗: %w", promptErr)
		return nil, err
	}

	// 3. Gemini API を呼び出し
	slog.Info("ScriptRunner: Calling Gemini API", "model", sr.cfg.GeminiModel)
	resp, err := sr.aiClient.GenerateContent(ctx, finalPrompt, sr.cfg.GeminiModel)
	if err != nil {
		return nil, fmt.Errorf("プロンプト生成に失敗: %w", err)
	}

	// 4. AI 応答をパースして構造化データに変換
	manga, err := sr.parseResponse(resp.Text)
	if err != nil {
		return nil, err
	}

	return manga, nil
}

// extractContent 抽出機能を使用して指定された URL からテキスト コンテンツを抽出し、コンテンツまたはエラーを返します。
func (sr *MangaScriptRunner) extractContent(ctx context.Context, url string) (string, error) {
	text, _, err := sr.extractor.FetchAndExtractText(ctx, url)
	if err != nil {
		return "", fmt.Errorf("failed to extract text from URL: %w", err)
	}
	return text, nil
}

// parseResponse AI API からの生の JSON 応答を取得し、解析されたデータを返します。
func (sr *MangaScriptRunner) parseResponse(raw string) (*domain.MangaResponse, error) {
	raw = strings.TrimSpace(raw)
	var rawJSON string

	matches := jsonBlockRegex.FindStringSubmatch(raw)
	if len(matches) > 1 {
		rawJSON = matches[1]
	} else {
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
		return nil, fmt.Errorf("AIからの応答に含まれるJSONの解析に失敗しました (応答抜粋: %q): %w", truncateString(raw, 200), err)
	}

	return &manga, nil
}

// truncateString 文字列を指定された最大長に切り捨てます。
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
