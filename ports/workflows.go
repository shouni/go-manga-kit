package ports

import (
	"context"

	imagePorts "github.com/shouni/gemini-image-kit/ports"
)

// Workflows は、構築済みの各 Runner を保持します。
type Workflows struct {
	Design     DesignRunner
	Script     ScriptRunner
	PanelImage PanelImageRunner
	PageImage  PageImageRunner
	Publish    PublishRunner
	CloseFunc  func()
}

// Close は、保持しているリソースの解放関数（CloseFunc）を呼び出します。
func (w *Workflows) Close() {
	if w != nil && w.CloseFunc != nil {
		w.CloseFunc()
	}
}

// DesignRunner は、キャラクターIDに基づいてデザインシートを生成し、Seed値を特定する責務を持ちます。
// aspectRatio・layoutKind・override は runner.MangaDesignRunner.Run のドキュメントを参照して
// ください。
type DesignRunner interface {
	Run(ctx context.Context, charIDs []string, seed int64, outputDir, aspectRatio, layoutKind string, override DesignOverride) (string, int64, error)
}

// DesignOverride は、Run の1回の呼び出しに限定して、キャラクターの参照画像・visual_cuesを
// 差し替えるためのその場限りの上書き指定です。go-character-kit 側のキャラクター定義
// （characters.json）そのものは変更しません。ReferenceURL/VisualCues が空の場合はそのフィール
// ドのみキャラクター定義の値を使います。charIDs が複数（合成デザインシート）の場合、上書きは
// どのキャラクターに適用すべきか一意に決まらないため無視されます。
type DesignOverride struct {
	ReferenceURL string
	VisualCues   []string
}

// ScriptRunner は、ソース（URLやテキスト）を解析し、構造化された漫画台本を生成する責務を持ちます。
type ScriptRunner interface {
	Run(ctx context.Context, scriptURL string, mode string) (*MangaResponse, error)
}

// PanelImageRunner は、解析済みの漫画データと対象パネルのインデックスを基に、パネル画像を生成する責務を持ちます。
type PanelImageRunner interface {
	Run(ctx context.Context, manga *MangaResponse) ([]*imagePorts.ImageResponse, error)
	RunAndSave(ctx context.Context, manga *MangaResponse, outputPath string) (*MangaResponse, error)
}

// PageImageRunner は、解析済みの漫画データから漫画のページ画像を生成する責務を持ちます。
type PageImageRunner interface {
	Run(ctx context.Context, manga *MangaResponse) ([]*imagePorts.ImageResponse, error)
	RunAndSave(ctx context.Context, manga *MangaResponse, outputPath string) ([]string, error)
}

// PublishRunner は、漫画データを統合し、指定された形式（例: HTML）で出力する責務を持ちます。
type PublishRunner interface {
	Run(ctx context.Context, manga *MangaResponse, outputDir string) (*PublishResult, error)
	BuildMarkdown(manga *MangaResponse) string
}
