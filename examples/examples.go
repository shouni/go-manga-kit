package examples

import (
	"context"
	_ "embed"
	"encoding/json"

	"github.com/shouni/go-manga-kit/pkg/domain"

	"github.com/shouni/go-remote-io/pkg/gcsfactory"
)

//go:embed characters.json
var CharactersJSON []byte

// LoadMangaScript は指定されたパス（JSON）を読み込み、ドメインモデルに変換するのだ
func LoadMangaScript(ctx context.Context, rioFactory gcsfactory.Factory) (*domain.MangaResponse, error) {
	reader, _ := rioFactory.NewInputReader()
	rc, err := reader.Open(ctx, "examples/manga_script.json")
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var manga domain.MangaResponse
	if err := json.NewDecoder(rc).Decode(&manga); err != nil {
		return nil, err
	}
	return &manga, nil
}
