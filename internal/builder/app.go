package builder

import (
	"github.com/shouni/go-manga-kit/internal/config"
	"github.com/shouni/go-manga-kit/pkg/generator"

	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

// AppContext は、アプリケーション実行に必要な共通コンテキストを保持する
// これを各Build関数に渡すことで、依存関係の注入を簡素化します。
type AppContext struct {
	Config         *config.Config           // Config: 環境変数から読み込まれたグローバルな設定です（APIキー、プロジェクトIDなど）。
	Options        config.GenerateOptions   // Options: コマンドラインから渡された実行時の設定です（モード、URL、モデル名など）。
	Reader         remoteio.InputReader     // Reader: 外部データやスクリプトの読み込みに使用する入力元です。
	Writer         remoteio.OutputWriter    // Writer: 生成された内容を保存するための出力先です。
	MangaGenerator generator.MangaGenerator // MangaGenerator: 画像生成とキャラクター管理を含むマンガ生成のコア機能です。
	aiClient       gemini.GenerativeModel   // aiClient: Geminiの通信に使う共通クライアント
	httpClient     httpkit.ClientInterface  // httpClient: 外部APIとの通信に使う共通クライアント
}

// NewAppContext は AppContext の新しいインスタンスを生成する
func NewAppContext(
	cfg *config.Config,
	httpClient httpkit.ClientInterface,
	aiClient gemini.GenerativeModel,
	reader remoteio.InputReader,
	writer remoteio.OutputWriter,
	mangaGenerator generator.MangaGenerator,
) AppContext {
	return AppContext{
		Config:         cfg,
		Options:        cfg.Options,
		aiClient:       aiClient,
		httpClient:     httpClient,
		Reader:         reader,
		Writer:         writer,
		MangaGenerator: mangaGenerator,
	}
}
