package publisher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/shouni/go-manga-kit/ports"
)

// --- Mocks ---

type mockWriter struct {
	files map[string][]byte
}

func (m *mockWriter) Write(ctx context.Context, path string, content io.Reader, contentType string) error {
	data, err := io.ReadAll(content)
	if err != nil {
		return err
	}
	if m.files == nil {
		m.files = make(map[string][]byte)
	}
	m.files[path] = data
	return nil
}

type mockMDRunner struct {
	runCalled bool
}

// 戻り値を *bytes.Buffer に修正し、インターフェースに合わせる
func (m *mockMDRunner) Run(title string, markdown []byte) (*bytes.Buffer, error) {
	m.runCalled = true
	html := fmt.Sprintf("<html><head><title>%s</title></head><body>%s</body></html>", title, string(markdown))
	return bytes.NewBufferString(html), nil
}

// --- Tests ---

func TestMangaPublisher_BuildMarkdown(t *testing.T) {
	p := NewMangaPublisher(nil, nil)

	manga := &ports.MangaResponse{
		Title:       "テスト漫画",
		Description: "説明文",
		Panels: []ports.Panel{
			{
				SpeakerID:    "zundamon",
				Dialogue:     "こんにちは <なのだ>", // エスケープのテスト
				VisualAnchor: "微笑むずんだもん",
				ReferenceURL: "gs://bucket/p1.png",
			},
		},
	}

	opts := ports.PublishOptions{
		ImagePaths: []string{"images/p1.png"},
	}

	got := p.BuildMarkdown(manga, opts)

	tests := []string{
		"# テスト漫画",
		"説明文",
		"![Panel 1](images/p1.png)",
		"**zundamon**: こんにちは &lt;なのだ&gt;",
		"> **Visual Anchor:** 微笑むずんだもん",
	}

	for _, want := range tests {
		if !strings.Contains(got, want) {
			t.Errorf("Markdown missing expected content: %q\nGot:\n%s", want, got)
		}
	}
}

func TestMangaPublisher_Publish(t *testing.T) {
	ctx := context.Background()
	writer := &mockWriter{files: make(map[string][]byte)}
	mdRunner := &mockMDRunner{}
	p := NewMangaPublisher(writer, mdRunner)

	manga := &ports.MangaResponse{
		Title: "成果物保存テスト",
		Panels: []ports.Panel{
			{
				SpeakerID:    "metan",
				Dialogue:     "保存しますわよ。",
				ReferenceURL: "gs://bucket/metan_01.png",
			},
		},
	}

	opts := ports.PublishOptions{
		OutputDir: "gs://my-output/result/",
	}

	result, err := p.Publish(ctx, manga, opts)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// 1. パスの解決確認
	// 実際に出力された "manga_plot.md" に合わせる
	expectedMDName := "manga_plot.md"
	expectedHTMLName := "manga_plot.html"

	if !strings.HasSuffix(result.MarkdownPath, expectedMDName) {
		t.Errorf("Unexpected Markdown path: %s (expected suffix: %s)", result.MarkdownPath, expectedMDName)
	}
	if !strings.HasSuffix(result.HTMLPath, expectedHTMLName) {
		t.Errorf("Unexpected HTML path: %s (expected suffix: %s)", result.HTMLPath, expectedHTMLName)
	}

	// 2. 書き込み確認
	if _, ok := writer.files[result.MarkdownPath]; !ok {
		t.Errorf("Markdown file %s was not written to writer", result.MarkdownPath)
	}
	if _, ok := writer.files[result.HTMLPath]; !ok {
		t.Errorf("HTML file %s was not written to writer", result.HTMLPath)
	}

	// 3. MD Runner が呼ばれたか
	if !mdRunner.runCalled {
		t.Error("MD runner was not called")
	}

	// 4. 画像パスの相対化確認
	expectedImgPath := "images/metan_01.png"
	if len(result.ImagePaths) == 0 || result.ImagePaths[0] != expectedImgPath {
		t.Errorf("Expected image path %s, got %v", expectedImgPath, result.ImagePaths)
	}
}

func TestMangaPublisher_Publish_NilManga(t *testing.T) {
	p := NewMangaPublisher(nil, nil)
	_, err := p.Publish(context.Background(), nil, ports.PublishOptions{})
	if err == nil {
		t.Error("Expected error when manga is nil, got nil")
	}
}
