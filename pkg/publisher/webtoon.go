package publisher

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/shouni/go-text-format/pkg/domain"
	"github.com/shouni/go-text-format/pkg/md2html"
	"github.com/shouni/go-text-format/pkg/renderer"
)

// WebtoonPublisher は go-text-format の機能を活用し、
// 構造化データから高品質な Webtoon ビューアを生成するパブリッシャーです。
type WebtoonPublisher struct {
	renderer *renderer.WebtoonRenderer
}

// NewWebtoonPublisher は、Markdown コンバーターを受け取り、
// セリフのパース設定を引き継いだレンダラーを初期化して返します。
func NewWebtoonPublisher(conv md2html.Converter) (*WebtoonPublisher, error) {
	// 記憶した go-text-format の NewWebtoonRenderer を使用
	r, err := renderer.NewWebtoonRenderer(conv)
	if err != nil {
		return nil, fmt.Errorf("webtoon_publisher: レンダラーの初期化に失敗しました: %w", err)
	}

	return &WebtoonPublisher{
		renderer: r,
	}, nil
}

// Publish は go-text-format のドメインモデルを受け取り、
// 指定された io.Writer（ファイルやバッファ）に完全な HTML をレンダリングします。
func (wp *WebtoonPublisher) Publish(ctx context.Context, w io.Writer, webtoon *domain.Webtoon, lang string) error {
	if webtoon == nil {
		return fmt.Errorf("webtoon_publisher: データが空です")
	}

	if lang == "" {
		lang = "ja" // デフォルト言語
	}

	slog.Info("Webtoon HTMLのレンダリングを開始しますなのだ",
		"title", webtoon.Title,
		"panels", len(webtoon.Panels),
	)

	// go-text-format の RenderWebtoon を呼び出し、
	// 内蔵の webtoon.html テンプレートと webtoon.css を適用します。
	if err := wp.renderer.RenderWebtoon(w, webtoon, lang); err != nil {
		return fmt.Errorf("webtoon_publisher: HTML生成に失敗しました: %w", err)
	}

	return nil
}

// CreateWebtoonModel は、解析・演出済みのパネルデータから
// go-text-format が解釈可能な Webtoon ドメインモデルを構築します。
func (wp *WebtoonPublisher) CreateWebtoonModel(title string, panels []domain.Panel) *domain.Webtoon {
	return &domain.Webtoon{
		Title:  title,
		Panels: panels,
	}
}
