package runner

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	imgdom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/generator"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

const designPromptTemplate = "Masterpiece character design sheet of %d characters: %s. Each character is distinct, side-by-side, multiple views (front, side, back), standing full body"

// MangaDesignRunner はキャラクターデザインシート生成の実行実体なのだ。
type MangaDesignRunner struct {
	cfg      config.Config
	mangaGen generator.MangaGenerator
	writer   remoteio.OutputWriter
}

// NewMangaDesignRunner は依存関係を注入して初期化するのだ。
func NewMangaDesignRunner(cfg config.Config, mangaGen generator.MangaGenerator, writer remoteio.OutputWriter) *MangaDesignRunner {
	return &MangaDesignRunner{
		cfg:      cfg,
		mangaGen: mangaGen,
		writer:   writer,
	}
}

// Run は、キャラクターIDを指定してデザインシートを生成し、GCSやローカルに保存するのだ。
func (dr *MangaDesignRunner) Run(ctx context.Context, charIDs []string, seed int64, outputGCS string) (string, int64, error) {
	// 1. 複数キャラの情報を集約 (CharactersMap などから取得)
	refs, descriptions, err := collectCharacterAssets(dr.mangaGen.Characters, charIDs)
	if err != nil {
		return "", 0, fmt.Errorf("キャラクター資産の収集に失敗しました: %w", err)
	}

	slog.Info("Executing design work generation",
		slog.Any("chars", charIDs),
		slog.Int("ref_count", len(refs)),
	)

	// 2. プロンプト構築
	designPrompt := dr.buildDesignPrompt(descriptions)

	// 3. 生成リクエスト (Seedが0の場合はランダム生成)
	pageReq := imgdom.ImagePageRequest{
		Prompt:        designPrompt,
		ReferenceURLs: refs,
		AspectRatio:   "16:9",
		Seed:          ptrInt64(seed),
	}

	// 4. 生成実行 (Gemini Nano Banana 呼び出し)
	resp, err := dr.mangaGen.ImgGen.GenerateMangaPage(ctx, pageReq)
	if err != nil {
		slog.Error("Design generation failed", "error", err)
		return "", 0, fmt.Errorf("画像の生成に失敗しました: %w", err)
	}

	// 5. 画像の保存 (GCS: OutputGCS を優先的に使用)
	outputPath, err := dr.saveResponseImage(ctx, *resp, charIDs, outputGCS)
	if err != nil {
		slog.Error("Failed to save image", "error", err)
		return "", 0, fmt.Errorf("画像の保存に失敗しました: %w", err)
	}

	return outputPath, resp.UsedSeed, nil
}

func (dr *MangaDesignRunner) saveResponseImage(ctx context.Context, resp imgdom.ImageResponse, charIDs []string, imageDir string) (string, error) {
	extension := getPreferredExtension(resp.MimeType)
	charTags := strings.Join(charIDs, "_")
	filename := fmt.Sprintf("design_%s%s", charTags, extension)
	var finalPath string
	var err error

	if remoteio.IsRemoteURI(imageDir) {
		finalPath, err = url.JoinPath(imageDir, filename)
	} else {
		finalPath = filepath.Join(imageDir, filename)
		if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
			return "", err
		}
	}

	if err != nil {
		return "", err
	}

	if err := dr.writer.Write(ctx, finalPath, bytes.NewReader(resp.Data), resp.MimeType); err != nil {
		return "", err
	}

	return finalPath, nil
}

// buildDesignPrompt キャラクターデザインシートを生成するための詳細なプロンプト文字列を構築します。
func (dr *MangaDesignRunner) buildDesignPrompt(descriptions []string) string {
	base := fmt.Sprintf(designPromptTemplate, len(descriptions), strings.Join(descriptions, " and "))

	// プロンプトの各要素をスライスに集約
	promptParts := []string{base}
	if dr.cfg.StyleSuffix != "" {
		promptParts = append(promptParts, dr.cfg.StyleSuffix)
	}
	promptParts = append(promptParts, "white background", "sharp focus", "4k resolution")

	// カンマとスペースで結合することで、空の要素による不正な出力を防ぐ
	return strings.Join(promptParts, ", ")
}

// collectCharacterAssets CharactersMap から、指定されたキャラクター ID の参照 URL と説明を取得します。
// 有効な参照が見つからない場合は、エラーとともに参照 URL と説明のスライスを返します。
func collectCharacterAssets(chars domain.CharactersMap, ids []string) ([]string, []string, error) {
	var referenceURLs []string
	var descriptions []string
	var missingIDs []string
	processedIDs := make(map[string]struct{})

	for _, id := range ids {
		if _, exists := processedIDs[id]; exists {
			continue
		}
		processedIDs[id] = struct{}{}

		char := chars.FindCharacter(id)
		if char == nil {
			missingIDs = append(missingIDs, id)
			continue
		}

		if char.ReferenceURL != "" {
			referenceURLs = append(referenceURLs, char.ReferenceURL)
		}
		desc := char.Name
		if len(char.VisualCues) > 0 {
			desc = fmt.Sprintf("%s (%s)", char.Name, strings.Join(char.VisualCues, ", "))
		}
		descriptions = append(descriptions, desc)
	}

	if len(missingIDs) > 0 {
		return nil, nil, fmt.Errorf("指定されたキャラクターIDが見つかりませんでした: %s", strings.Join(missingIDs, ", "))
	}

	if len(referenceURLs) == 0 {
		return nil, nil, fmt.Errorf("有効な参照URLを持つキャラクターが1つも見つかりませんでした (対象ID: %s)", strings.Join(ids, ", "))
	}
	return referenceURLs, descriptions, nil
}

func ptrInt64(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}

func getPreferredExtension(mimeType string) string {
	preferred := map[string]string{"image/png": ".png", "image/jpeg": ".jpg"}
	if ext, ok := preferred[mimeType]; ok {
		return ext
	}
	return ".png"
}
