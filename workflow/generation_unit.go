package workflow

import (
	"fmt"

	"github.com/shouni/gemini-image-kit/generator"
	"github.com/shouni/go-gemini-client/gemini"

	"github.com/shouni/go-manga-kit/layout"
	"github.com/shouni/go-manga-kit/ports"
)

// buildGenerationUnit は、特定の AI クライアントとモデル設定に基づき、 core, composer, generator をひとまとめにした LLM 構造体を構築します。
func (m *manager) buildGenerationUnit(client gemini.GenerativeModel, modelName string) (*generationUnit, error) {
	cache := newImageCache(defaultCacheExpiration)

	core, err := m.buildCore(client, cache)
	if err != nil {
		cache.Stop()
		return nil, err
	}

	composer, err := m.buildComposer(core, m.promptDeps.Characters)
	if err != nil {
		cache.Stop()
		return nil, err
	}

	gen, err := m.buildGenerator(core)
	if err != nil {
		cache.Stop()
		return nil, err
	}
	cache.Start()

	return &generationUnit{
		imageGenerator: gen,
		mangaComposer:  composer,
		model:          modelName,
		cache:          cache,
	}, nil
}

// buildCore はGeminiImageCoreエンジンを初期化します。
func (m *manager) buildCore(aiClient gemini.GenerativeModel, cache *imageCache) (*generator.GeminiImageCore, error) {
	core, err := generator.NewGeminiImageCore(
		aiClient,
		m.reader,
		m.httpClient,
		cache,
		defaultTTL,
		false,
	)
	if err != nil {
		return nil, fmt.Errorf("画像生成エンジンの初期化に失敗しました: %w", err)
	}

	return core, nil
}

// buildComposer は提供された構成と依存関係を使用して MangaComposerインスタンスを初期化し、返します。
func (m *manager) buildComposer(
	core *generator.GeminiImageCore,
	chars *ports.Characters,
) (*layout.MangaComposer, error) {
	composer, err := layout.NewMangaComposer(
		core,
		core,
		chars,
	)
	if err != nil {
		return nil, fmt.Errorf("MangaComposerの初期化に失敗しました: %w", err)
	}

	return composer, nil
}

// buildGenerator は提供された構成と依存関係を使用して ImageGenerator インスタンスを初期化し、返します。
func (m *manager) buildGenerator(core *generator.GeminiImageCore) (*generator.GeminiGenerator, error) {
	gen, err := generator.NewGeminiGenerator(core)
	if err != nil {
		return nil, fmt.Errorf("GeminiGeneratorの初期化に失敗しました: %w", err)
	}

	return gen, nil
}
