package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/shouni/go-manga-kit/examples"
	"github.com/shouni/go-manga-kit/pkg/builder"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/pipeline"

	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/gcsfactory"
)

const (
	DefaultImageModel        = "gemini-3-pro-image-preview"
	DefaultHTTPTimeout       = 30 * time.Second
	DefaultImagePromptSuffix = "Japanese anime style, official art, cel-shaded, clean line art, high-quality manga coloring, expressive eyes, vibrant colors, cinematic lighting, masterpiece, ultra-detailed, flat shading, clear character features, no 3D effect, high resolution"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: apmango [image|story]")
		os.Exit(1)
	}

	subcommand := os.Args[1]
	ctx := context.Background()
	setupLogger()

	// --- 1. インフラ・共通リソースの初期化 ---
	rioFactory, err := gcsfactory.NewGCSClientFactory(ctx)
	if err != nil {
		log.Fatal("Factory初期化失敗なのだ: ", err)
	}

	aiClient, err := builder.InitializeAIClient(ctx, os.Getenv("GENAI_API_KEY"))
	if err != nil {
		log.Fatal("AIクライアント初期化失敗なのだ: ", err)
	}

	httpClient := httpkit.New(DefaultHTTPTimeout)
	imgCore := builder.InitializeImageCore(httpClient)

	// --- 2. データのロード (JSON & DNA) ---
	manga, err := examples.LoadMangaScript(ctx, rioFactory)
	if err != nil {
		log.Fatal("台本ロード失敗なのだ: ", err)
	}

	// DNAデータを取得するのだ
	chars, err := domain.GetCharacters(examples.CharactersJSON)
	if err != nil {
		log.Fatal("DNAデータロード失敗なのだ: ", err)
	}

	// DNAデータを取得するのだ
	charMap := domain.BuildCharactersMap(chars)
	if err != nil {
		log.Fatal("DNAデータロード失敗なのだ: ", err)
	}

	// DNAデータが揃ってから PromptBuilder を初期化するのだ！
	promptBuilder := builder.NewPromptBuilder(charMap, DefaultImagePromptSuffix)

	// 汎用ライターの取得
	writer, _ := rioFactory.NewOutputWriter()

	// --- 3. サブコマンド分岐 ---
	switch subcommand {
	case "image":
		slog.Info("Phase 2: 個別パネル生成モード開始なのだ", "count", len(manga.Pages))

		adapter, err := builder.InitializeImageAdapter(imgCore, aiClient, DefaultImageModel, DefaultImagePromptSuffix)
		if err != nil {
			log.Fatal(err)
		}

		// パイプライン実行
		pipe := pipeline.NewIndividualPipeline(promptBuilder, adapter)
		results, err := pipe.Execute(ctx, manga)
		if err != nil {
			log.Fatal("個別生成失敗なのだ: ", err)
		}

		// 保存処理
		for i, resp := range results {
			path := fmt.Sprintf("output/panels/panel_%d.png", manga.Pages[i].Page)
			content := bytes.NewReader(resp.Data)
			if err := writer.Write(ctx, path, content, "image/png"); err != nil {
				slog.Error("保存失敗", "path", path, "error", err)
				continue
			}
			slog.Info("保存完了なのだ！", "path", path)
		}

	case "story":
		slog.Info("Phase 3: 最終ページ統合モード開始なのだ")

		adapter := builder.InitializeMangaPageAdapter(imgCore, aiClient, DefaultImageModel)
		pipe := pipeline.NewPagePipeline(promptBuilder, adapter)

		// PagePipeline の Execute は漫画データとDNAスライスの両方を受け取る設計なのだ
		result, err := pipe.Execute(ctx, manga, chars)
		if err != nil {
			log.Fatal("ページ統合生成失敗なのだ: ", err)
		}

		// 最終成果物の保存
		finalPath := "output/final_manga_page.png"
		content := bytes.NewReader(result.Data)
		if err := writer.Write(ctx, finalPath, content, "image/png"); err != nil {
			log.Fatal("最終保存失敗なのだ: ", err)
		}
		slog.Info("全工程終了！最高の一枚が完成したのだ！", "path", finalPath)

	default:
		fmt.Printf("未知のコマンドなのだ: %s\n", subcommand)
		os.Exit(1)
	}
}

func setupLogger() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
}
