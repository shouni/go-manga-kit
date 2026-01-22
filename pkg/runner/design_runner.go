package runner

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/asset"
	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/generator"

	imgdom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

const (
	// プロンプト構成用の定数
	designPromptBaseTemplate = "Masterpiece character design sheet of %s"
	designLayoutDefault      = "multiple views (front, side, back), standing full body"
	designLayoutPromptFormat = "Layout: %s, side-by-side, separate character charts"
)

// fileNameSanitizer はファイル名として使用できない文字を置換します。
var fileNameSanitizer = strings.NewReplacer(
	"/", "_",
	`\`, "_",
	":", "_",
	"*", "_",
	"?", "_",
	`"`, "_",
	"<", "_",
	">", "_",
	"|", "_",
)

// MangaDesignRunner はキャラクターデザインシート生成の実行実体なのだ。
type MangaDesignRunner struct {
	cfg      config.Config
	composer *generator.MangaComposer
	writer   remoteio.OutputWriter
}

// NewMangaDesignRunner は依存関係を注入して初期化します。
func NewMangaDesignRunner(cfg config.Config, composer *generator.MangaComposer, writer remoteio.OutputWriter) *MangaDesignRunner {
	return &MangaDesignRunner{
		cfg:      cfg,
		composer: composer,
		writer:   writer,
	}
}

// Run は、指定されたキャラクターIDのデザインシートを生成し、指定されたディレクトリに保存します。
func (dr *MangaDesignRunner) Run(ctx context.Context, charIDs []string, seed int64, outputDir string) (string, int64, error) {
	// 1. 複数キャラの情報を集約
	refs, descriptions, err := collectCharacterAssets(dr.composer.CharactersMap, charIDs)
	if err != nil {
		return "", 0, fmt.Errorf("キャラクター資産の収集に失敗しました: %w", err)
	}

	slog.Info("Executing design work generation",
		slog.Any("chars", charIDs),
		slog.Int("ref_count", len(refs)),
	)

	// 2. プロンプト構築
	designPrompt := dr.buildDesignPrompt(descriptions)
	if designPrompt == "" {
		return "", 0, fmt.Errorf("キャラクター情報が空のため、プロンプトを生成できませんでした")
	}

	// 3. 生成リクエスト
	pageReq := imgdom.ImagePageRequest{
		Prompt:        designPrompt,
		ReferenceURLs: refs,
		AspectRatio:   "16:9",
		Seed:          ptrInt64(seed),
	}

	// 4. 生成実行
	resp, err := dr.composer.ImageGenerator.GenerateMangaPage(ctx, pageReq)
	if err != nil {
		slog.Error("Design generation failed", "error", err)
		return "", 0, fmt.Errorf("画像の生成に失敗しました: %w", err)
	}

	// 5. 画像の保存
	outputPath, err := dr.saveResponseImage(ctx, *resp, charIDs, outputDir)
	if err != nil {
		slog.Error("Failed to save image", "error", err)
		return "", 0, fmt.Errorf("画像の保存に失敗しました: %w", err)
	}

	return outputPath, resp.UsedSeed, nil
}

// saveResponseImage は、生成された画像データを指定されたディレクトリに保存します。
func (dr *MangaDesignRunner) saveResponseImage(ctx context.Context, resp imgdom.ImageResponse, charIDs []string, outputDir string) (string, error) {
	charTags := strings.Join(charIDs, "_")
	sanitizedCharTags := fileNameSanitizer.Replace(charTags)

	extension := getPreferredExtension(resp.MimeType)
	filename := fmt.Sprintf("design_%s%s", sanitizedCharTags, extension)
	finalPath, err := asset.ResolveOutputPath(outputDir, path.Join(asset.CharacterDesignDir, filename))
	if err != nil {
		return "", fmt.Errorf("画像保存パスの生成に失敗しました (path: %s): %w", finalPath, err)
	}

	if err = dr.writer.Write(ctx, finalPath, bytes.NewReader(resp.Data), resp.MimeType); err != nil {
		return "", fmt.Errorf("画像の保存に失敗しました (path: %s): %w", finalPath, err)
	}

	return finalPath, nil
}

// buildDesignPrompt キャラクターデザインシートを生成するための詳細なプロンプト文字列を構築します。
func (dr *MangaDesignRunner) buildDesignPrompt(descriptions []string) string {
	numChars := len(descriptions)
	if numChars == 0 {
		slog.Warn("buildDesignPrompt called with empty descriptions")
		return ""
	}

	var subjects string
	if numChars > 1 {
		subjectParts := make([]string, numChars)
		for i, d := range descriptions {
			subjectParts[i] = fmt.Sprintf("[Subject %d: %s]", i+1, d)
		}
		subjects = fmt.Sprintf("%d DIFFERENT characters: %s", numChars, strings.Join(subjectParts, " "))
	} else {
		subjects = descriptions[0]
	}

	base := fmt.Sprintf(designPromptBaseTemplate, subjects)
	layout := fmt.Sprintf(designLayoutPromptFormat, designLayoutDefault)

	promptParts := []string{base, layout}
	if dr.cfg.StyleSuffix != "" {
		promptParts = append(promptParts, dr.cfg.StyleSuffix)
	}
	promptParts = append(promptParts, "white background", "sharp focus", "4k resolution")

	return strings.Join(promptParts, ", ")
}

// collectCharacterAssets キャラクター情報を収集し、参照URLと説明文を返します。
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

		char := chars.GetCharacter(id)
		if char == nil {
			missingIDs = append(missingIDs, id)
			continue
		}

		if char.ReferenceURL == "" {
			slog.Warn("キャラクターに参照URLがないためスキップします", "id", id)
			continue
		}

		referenceURLs = append(referenceURLs, char.ReferenceURL)

		desc := char.Name
		if len(char.VisualCues) > 0 {
			desc = fmt.Sprintf("%s (%s)", char.Name, strings.Join(char.VisualCues, ", "))
		}
		descriptions = append(descriptions, desc)
	}

	if len(missingIDs) > 0 {
		return nil, nil, fmt.Errorf("一部のキャラクターIDが見つかりませんでした: %s", strings.Join(missingIDs, ", "))
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
