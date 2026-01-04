package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/internal/config"
	mngdom "github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"github.com/shouni/go-text-format/pkg/md2htmlrunner"
)

// スタイルおよびデフォルト値の定数定義
const (
	evenPanelTail        = "top"
	evenPanelBottom      = "10%"
	evenPanelLeft        = "10%"
	oddPanelTail         = "bottom"
	oddPanelTop          = "10%"
	oddPanelRight        = "10%"
	placeholderImageName = "placeholder.png"
	defaultNarrationName = "narration"
	markdownImageDir     = "images" // Markdown内で参照する画像ディレクトリ名
)

// tagRegex は、"[tag]" 形式のメタデータを抽出・除去するための正規表現です。
var tagRegex = regexp.MustCompile(`\[[^\]]+\]`)

// PublisherRunner は、生成された漫画の台本と画像を永続化（保存）するためのインターフェースなのだ
type PublisherRunner interface {
	Run(ctx context.Context, manga mngdom.MangaResponse, images []*imagedom.ImageResponse) error
}

// DefaultPublisherRunner は、PublisherRunner の標準的な実装を提供する構造体なのだ
type DefaultPublisherRunner struct {
	options    config.GenerateOptions
	writer     remoteio.OutputWriter
	htmlRunner md2htmlrunner.Runner
}

// NewDefaultPublisherRunner は、DefaultPublisherRunner の新しいインスタンスを生成するコンストラクタなのだ
func NewDefaultPublisherRunner(options config.GenerateOptions, writer remoteio.OutputWriter, htmlRunner md2htmlrunner.Runner) *DefaultPublisherRunner {
	return &DefaultPublisherRunner{
		options:    options,
		writer:     writer,
		htmlRunner: htmlRunner,
	}
}

// Run はパブリッシュ工程のメインフローを制御するのだ
func (pr *DefaultPublisherRunner) Run(ctx context.Context, manga mngdom.MangaResponse, images []*imagedom.ImageResponse) error {
	// 1. 画像の保存
	imagePaths, err := pr.saveImages(ctx, images)
	if err != nil {
		return fmt.Errorf("画像の保存に失敗したのだ: %w", err)
	}

	// Markdownに書き込むパスを、ディレクトリ名を含まない相対パスに変換するのだ
	relativeImagePaths := make([]string, 0, len(imagePaths))
	for _, path := range imagePaths {
		relPath := filepath.Join(markdownImageDir, filepath.Base(path))
		relativeImagePaths = append(relativeImagePaths, relPath)
	}

	// 2. Markdownを構築
	finalContent := pr.buildFinalMarkdown(manga, relativeImagePaths)

	// 3. スクリプト（Markdown）の保存
	if err := pr.saveMangaScript(ctx, finalContent); err != nil {
		return err
	}

	// 4. HTMLへの変換と保存
	if pr.htmlRunner != nil {
		slog.Info("Webtoon HTMLへの変換を開始するのだ", "title", manga.Title)
		htmlBuffer, err := pr.htmlRunner.Run(ctx, manga.Title, []byte(finalContent))
		if err != nil {
			return fmt.Errorf("HTMLへの変換に失敗したのだ: %w", err)
		}

		htmlPath := strings.TrimSuffix(pr.options.OutputFile, filepath.Ext(pr.options.OutputFile)) + ".html"
		if err := pr.writer.Write(ctx, htmlPath, htmlBuffer, "text/html; charset=utf-8"); err != nil {
			return fmt.Errorf("%s へのHTML書き込みに失敗したのだ: %w", htmlPath, err)
		}
	}

	slog.Info("すべてのパブリッシュ工程が完了したのだ！")
	return nil
}

// buildFinalMarkdown は Markdownテキストを生成するのだ
func (pr *DefaultPublisherRunner) buildFinalMarkdown(manga mngdom.MangaResponse, imagePaths []string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", manga.Title))
	// マルチバイト文字対応: 話者名から一意なハッシュベースのクラス名を生成
	h := sha256.New()

	for i, page := range manga.Pages {
		imagePath := placeholderImageName
		if i < len(imagePaths) {
			imagePath = imagePaths[i]
		}

		sb.WriteString(fmt.Sprintf("## Panel: %s\n", imagePath))
		sb.WriteString("- layout: standard\n")

		if page.Dialogue != "" {
			// SpeakerIDを直接使用し、不安定なパース処理を削除
			speaker := page.SpeakerID
			if speaker == "" {
				speaker = defaultNarrationName
			}

			// dialogueからはタグをすべて除去してセリフ本文のみを抽出
			text := strings.TrimSpace(tagRegex.ReplaceAllString(page.Dialogue, ""))
			h.Reset()
			h.Write([]byte(speaker))
			hash := hex.EncodeToString(h.Sum(nil))
			speakerClass := "speaker-" + hash[:10]

			sb.WriteString(fmt.Sprintf("- speaker: %s\n", speakerClass))
			sb.WriteString(fmt.Sprintf("- text: %s\n", text))

			// 演出用スタイルの注入
			sb.WriteString(pr.getDialogueStyle(i))
		} else {
			sb.WriteString("- type: none\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// getDialogueStyle はパネルのインデックスに応じて吹き出しの配置スタイルを返すのだ
func (pr *DefaultPublisherRunner) getDialogueStyle(panelIndex int) string {
	if panelIndex%2 == 0 {
		return fmt.Sprintf("- tail: %s\n- bottom: %s\n- left: %s\n", evenPanelTail, evenPanelBottom, evenPanelLeft)
	}
	return fmt.Sprintf("- tail: %s\n- top: %s\n- right: %s\n", oddPanelTail, oddPanelTop, oddPanelRight)
}

// saveMangaScript は Markdown を保存するのだ
func (pr *DefaultPublisherRunner) saveMangaScript(ctx context.Context, content string) error {
	outputPath := pr.options.OutputFile
	slog.Info("Webtoonスクリプト(Markdown)を保存中...", "path", outputPath)
	reader := bytes.NewReader([]byte(content))
	return pr.writer.Write(ctx, outputPath, reader, "text/markdown; charset=utf-8")
}

// saveImages は画像群を保存するのだ
func (pr *DefaultPublisherRunner) saveImages(ctx context.Context, images []*imagedom.ImageResponse) ([]string, error) {
	baseDir := pr.options.OutputImageDir
	if baseDir == "" {
		baseDir = config.DefaultLocalImageDir
	}
	isGCS := remoteio.IsGCSURI(baseDir)

	var savedURIs []string
	for i, img := range images {
		if img == nil || len(img.Data) == 0 {
			continue
		}

		panelNumber := i + 1
		fileName := fmt.Sprintf("panel_%d.png", panelNumber)

		var fullPath string
		if isGCS {
			// クラウドストレージ用 (常にスラッシュ)
			var err error
			fullPath, err = url.JoinPath(baseDir, fileName)
			if err != nil {
				return nil, fmt.Errorf("GCSベースURIの解析に失敗: %w", err)
			}
		} else {
			// ローカルファイル用 (OS固有の区切り文字)
			fullPath = filepath.Join(baseDir, fileName)
		}

		if err := pr.writer.Write(ctx, fullPath, bytes.NewReader(img.Data), "image/png"); err != nil {
			return nil, fmt.Errorf("パネル %d (%s) の保存失敗: %w", panelNumber, fullPath, err)
		}
		savedURIs = append(savedURIs, fullPath)
	}
	slog.Info("すべての画像の保存が完了したのだ", "count", len(savedURIs))
	return savedURIs, nil
}
