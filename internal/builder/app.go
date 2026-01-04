package builder

import (
	"github.com/shouni/go-manga-kit/internal/config"
	"github.com/shouni/go-remote-io/pkg/remoteio"

	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
)

// AppContext は、アプリケーション実行に必要な共通コンテキストを保持するのだ。
// これを各 Build 関数に渡すことで、依存関係の整理を楽にするのだ！
type AppContext struct {
	// config は環境変数から読み込まれたグローバルな設定（APIキー、プロジェクトIDなど）なのだ
	Config *config.Config

	// options はコマンドラインから渡された実行時の設定（モード、URL、モデル名など）なのだ
	Options config.GenerateOptions

	Reader remoteio.InputReader
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
