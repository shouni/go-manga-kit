package builder

import (
	"github.com/shouni/go-manga-kit/internal/config"

	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/gcsfactory"
)

// AppContext は、アプリケーション実行に必要な共通コンテキストを保持するのだ。
// これを各 Build 関数に渡すことで、依存関係の整理を楽にするのだ！
type AppContext struct {
	// config は環境変数から読み込まれたグローバルな設定（APIキー、プロジェクトIDなど）なのだ
	Config *config.Config

	// options はコマンドラインから渡された実行時の設定（モード、URL、モデル名など）なのだ
	Options config.GenerateOptions

	// aiClient はGeminiの通信に使う共通クライアントなのだ
	aiClient gemini.GenerativeModel

	// httpClient は外部APIとの通信に使う共通クライアントなのだ
	httpClient httpkit.ClientInterface

	// remoteIOFactory は入力（Source）や出力（GCS/Local）を透過的に扱うためのファクトリなのだ
	RemoteIOFactory gcsfactory.Factory
}

// NewAppContext は AppContext の新しいインスタンスを生成するのだ
func NewAppContext(
	cfg *config.Config,
	aiClient gemini.GenerativeModel,
	httpClient httpkit.ClientInterface,
	rioFactory gcsfactory.Factory,
) AppContext {
	return AppContext{
		Config:          cfg,
		Options:         cfg.Options,
		aiClient:        aiClient,
		httpClient:      httpClient,
		RemoteIOFactory: rioFactory,
	}
}
