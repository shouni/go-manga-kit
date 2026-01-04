package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/shouni/go-manga-kit/internal/config"
	"github.com/shouni/go-manga-kit/internal/prompt"
	"github.com/shouni/go-manga-kit/pkg/domain"

	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"github.com/shouni/go-web-exact/v2/pkg/extract"
)

// ScriptRunner は、生成された漫画の台本や画像データを永続化するためのインターフェースなのだ。
type ScriptRunner interface {
	// Run は台本生成パイプラインを実行し、構造化された漫画データを返すのだ。
	Run(ctx context.Context) (domain.MangaResponse, error)
}

// MangaScriptRunner は、ドキュメントから漫画の構成案（台本）を生成する核となる構造体なのだ。
type MangaScriptRunner struct {
	cfg           config.Config          // 実行時のコマンドライン引数や設定
	extractor     *extract.Extractor     // Webサイトから本文を抽出するエクストラクター
	promptBuilder prompt.PromptBuilder   // AIに渡すプロンプトを構築するビルダー
	aiClient      gemini.GenerativeModel // Gemini APIと通信するクライアント
	reader        remoteio.InputReader   // ローカルやGCSのファイルを読み込むリーダー
}

// NewMangaScriptRunner は、MangaScriptRunnerの新しいインスタンスを生成して返すのだ。
func NewMangaScriptRunner(
	cfg config.Config,
	ext *extract.Extractor,
	pb prompt.PromptBuilder,
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

// Run は、入力ソースの読み込み、プロンプト構築、AIによる生成、結果のパースを一気に行うのだ。
func (sr *MangaScriptRunner) Run(ctx context.Context) (domain.MangaResponse, error) {
	// 1. 入力ソース（URL または ファイル）からテキストを読み込むのだ
	input, err := sr.readInputContent(ctx)
	if err != nil {
		return domain.MangaResponse{}, err
	}

	// 2. 読み取ったテキストをテンプレートに埋め込んでプロンプトを作るのだ
	promptContent, err := sr.promptBuilder.Build(prompt.TemplateData{InputText: string(input)})
	if err != nil {
		return domain.MangaResponse{}, err
	}

	// 3. Geminiを使って、漫画の構成案（JSON形式を期待）を生成させるのだ
	resp, err := sr.aiClient.GenerateContent(ctx, promptContent, sr.cfg.GeminiModel)
	if err != nil {
		return domain.MangaResponse{}, fmt.Errorf("台本の生成に失敗したのだ: %w", err)
	}

	// 4. AIが返したテキストからJSON部分を抽出し、構造体に変換するのだ
	manga, err := sr.parseResponse(resp.Text)
	if err != nil {
		return domain.MangaResponse{}, err
	}

	return manga, nil
}

// readInputContent は、URLまたはパスの設定に基づいて適切な方法でソースデータを取得するのだ。
func (sr *MangaScriptRunner) readInputContent(ctx context.Context) ([]byte, error) {
	// URLが指定されている場合は、Webスクレイピングを実行するのだ
	if sr.cfg.Options.ScriptURL != "" {
		text, _, err := sr.extractor.FetchAndExtractText(ctx, sr.cfg.Options.ScriptURL)
		return []byte(text), err
	}
	// ファイルパスが指定されている場合は、リーダーを使って開くのだ（GCS等も対応！）
	rc, err := sr.reader.Open(ctx, sr.cfg.Options.ScriptFile)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// parseResponse は、AIが返したテキストからMarkdownのコードブロック等を除去してJSONとしてパースするのだ。
func (sr *MangaScriptRunner) parseResponse(raw string) (domain.MangaResponse, error) {
	// 余計な空白や、AIが付けがちなMarkdownタグ (```json ... ```) を取り除く処理なのだ
	rawJSON := strings.TrimSpace(raw)
	rawJSON = strings.TrimPrefix(rawJSON, "```json")
	rawJSON = strings.TrimSuffix(rawJSON, "```")
	rawJSON = strings.TrimSpace(rawJSON)

	var manga domain.MangaResponse
	if err := json.Unmarshal([]byte(rawJSON), &manga); err != nil {
		return domain.MangaResponse{}, fmt.Errorf("JSONのパースに失敗したのだ: %w", err)
	}
	return manga, nil
}
