package workflow

import (
	"fmt"

	"github.com/shouni/go-gemini-client/pkg/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

// Builder は、ワークフローの各工程を担う Runner 群を構築・管理します。
type Builder struct {
	cfg config.Config
	//	chars      domain.CharactersMap
	httpClient    httpkit.ClientInterface
	aiClient      gemini.GenerativeModel
	reader        remoteio.InputReader
	writer        remoteio.OutputWriter
	mangaComposer *generator.MangaComposer
}

// NewBuilder は、設定とキャラクター定義を基に新しい Builder を初期化します。
func NewBuilder(cfg config.Config, httpClient httpkit.ClientInterface, aiClient gemini.GenerativeModel, reader remoteio.InputReader, writer remoteio.OutputWriter, charData []byte) (*Builder, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("httpClient は必須です")
	}
	if aiClient == nil {
		return nil, fmt.Errorf("aiClient は必須です")
	}
	if reader == nil {
		return nil, fmt.Errorf("reader は必須です")
	}
	if writer == nil {
		return nil, fmt.Errorf("writer は必須です")
	}
	mangaComposer, err := buildMangaComposer(cfg, httpClient, aiClient, reader, charData)
	if err != nil {
		return nil, fmt.Errorf("画像生成エンジンの初期化に失敗しました: %w", err)
	}

	return &Builder{
		cfg:           cfg,
		httpClient:    httpClient,
		aiClient:      aiClient,
		reader:        reader,
		writer:        writer,
		mangaComposer: mangaComposer,
	}, nil
}
