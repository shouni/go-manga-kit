package builder

import (
	"github.com/shouni/go-manga-kit/internal/config"

	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

// AppContext は、アプリケーション実行に必要な共通コンテキストを保持する
// これを各 Build 関数に渡すことで、依存関係の整理を楽にするのだ！
type AppContext struct {
	Config     *config.Config          // Config は環境変数から読み込まれたグローバルな設定（APIキー、プロジェクトIDなど）
	Options    config.GenerateOptions  // Options はコマンドラインから渡された実行時の設定（モード、URL、モデル名など）
	Reader     remoteio.InputReader    // Reader 外部データやスクリプトの読み込みに使用する、入力元（Reader）の設定
	Writer     remoteio.OutputWriter   // Writer 生成された内容を保存したり、外部へエクスポートしたりするための出力先を定義
	aiClient   gemini.GenerativeModel  // aiClient はGeminiの通信に使う共通クライアント
	httpClient httpkit.ClientInterface // httpClient は外部APIとの通信に使う共通クライアント
}

// NewAppContext は AppContext の新しいインスタンスを生成する
func NewAppContext(
	cfg *config.Config,
	httpClient httpkit.ClientInterface,
	aiClient gemini.GenerativeModel,
	reader remoteio.InputReader,
	writer remoteio.OutputWriter,
) AppContext {
	return AppContext{
		Config:     cfg,
		Options:    cfg.Options,
		aiClient:   aiClient,
		httpClient: httpClient,
		Reader:     reader,
		Writer:     writer,
	}
}
