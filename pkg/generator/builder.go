package generator

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/shouni/gemini-image-kit/pkg/generator"
	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-manga-kit/pkg/domain"
)

type Pipeline struct {
	ImgGen     generator.ImageGenerator
	Characters map[string]domain.Character
}

func NewPipeline(httpClient httpkit.ClientInterface, aiClient gemini.GenerativeModel, model, characterConfig string) (Pipeline, error) {
	characters, err := loadCharacters(characterConfig)
	if err != nil {
		return Pipeline{}, fmt.Errorf("loadCharacters failed: %w", err)
	}
	normalizedChars := make(map[string]domain.Character)
	for k, v := range characters {
		normalizedChars[strings.ToLower(k)] = v
	}

	imgGen, err := initializeImageGenerator(httpClient, aiClient, model)
	if err != nil {
		return Pipeline{}, fmt.Errorf("InitializeImageGenerator failed: %w", err)
	}

	return Pipeline{
		ImgGen:     imgGen,
		Characters: normalizedChars,
	}, nil
}

// initializeImageGenerator は ImageGeneratorを初期化します。
func initializeImageGenerator(httpClient httpkit.ClientInterface, aiClient gemini.GenerativeModel, model string) (generator.ImageGenerator, error) {
	imgCache := cache.New(30*time.Minute, 1*time.Hour)
	cacheTTL := 1 * time.Hour

	// 画像処理コアを生成
	core, err := generator.NewGeminiImageCore(
		httpClient,
		imgCache,
		cacheTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("GeminiImageCoreの初期化に失敗しました: %w", err)
	}

	imgGen, err := generator.NewGeminiGenerator(
		core,
		aiClient,
		model,
	)
	if err != nil {
		return nil, fmt.Errorf("GeminiGeneratorの初期化に失敗しました: %w", err)
	}

	return imgGen, nil
}

// loadCharacters は指定されたファイルパスからJSONを読み込み、キャラクターマップを返します。
func loadCharacters(path string) (map[string]domain.Character, error) {
	// 1. ファイルの読み込み
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("キャラクター設定ファイルの読み込みに失敗しました: %w", err)
	}

	// 2. バイト列からのパース処理（getCharacters）を再利用するのだ
	return domain.GetCharacters(data)
}
