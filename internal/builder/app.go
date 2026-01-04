package builder

import (
	"github.com/shouni/go-manga-kit/internal/config"

	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

// AppContext は、アプリケーション実行に必要な共通コンテキストを保持するのだ。
// これを各 Build 関数に渡すことで、依存関係の整理を楽にするのだ！
type AppContext struct {
	// config は環境変数から読み込まれたグローバルな設定（APIキー、プロジェクトIDなど）なのだ
	Config *config.Config

	// options はコマンドラインから渡された実行時の設定（モード、URL、モデル名など）なのだ
	Options config.GenerateOptions

	// Reader 外部データやスクリプトの読み込みに使用する、入力元（Reader）の設定
	Reader remoteio.InputReader

	// Writer 生成された内容を保存したり、外部へエクスポートしたりするための出力先を定義
	Writer remoteio.OutputWriter

	// aiClient はGeminiの通信に使う共通クライアントなのだ
	aiClient gemini.GenerativeModel

	// httpClient は外部APIとの通信に使う共通クライアントなのだ
	httpClient httpkit.ClientInterface
}

// NewAppContext は AppContext の新しいインスタンスを生成するのだ
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
