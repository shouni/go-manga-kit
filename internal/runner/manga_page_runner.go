package runner

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"
	mangakit "github.com/shouni/go-manga-kit/pkg/pipeline"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
)

var (
	titleRegex = regexp.MustCompile(`^#\s+(.+)`)
	panelRegex = regexp.MustCompile(`^##\s+Panel:?`)
	fieldRegex = regexp.MustCompile(`^\s*-\s*([a-zA-Z_]+):\s*(.+)`)
)

// PageRunner は MarkdownのパースとPipelineの実行を管理するのだ。
type PageRunner interface {
	Run(ctx context.Context, markdownContent string) (*imagedom.ImageResponse, error)
}

// MangaPageRunner は MarkdownのパースとPipelineの実行を管理するのだ。
type MangaPageRunner struct {
	pipeline *mangakit.PagePipeline
	baseURL  string
}

// NewMangaPageRunner initializes a MangaPageRunner with a manga pipeline, style suffix, and script URL.
func NewMangaPageRunner(mangaPipeline mangakit.Pipeline, styleSuffix string, scriptURL string) *MangaPageRunner {
	baseURL := ""
	u, err := url.Parse(scriptURL)
	if err == nil && u.Scheme == "gs" {
		baseURL = fmt.Sprintf("https://storage.googleapis.com/%s%s/", u.Host, path.Dir(u.Path))
	}

	return &MangaPageRunner{
		pipeline: mangakit.NewPagePipeline(mangaPipeline, styleSuffix), // manga全体を渡す
		baseURL:  baseURL,
	}
}

// Run processes the provided Markdown content and generates a manga page image using the configured pipeline.
func (r *MangaPageRunner) Run(ctx context.Context, markdownContent string) (*imagedom.ImageResponse, error) {
	manga, err := r.parseMarkdown(markdownContent)
	if err != nil {
		return nil, fmt.Errorf("Markdownのパースに失敗したのだ: %w", err)
	}
	return r.generateMangaPage(ctx, *manga)
}

// parseMarkdown parses the provided Markdown content and returns a MangaResponse object.
func (r *MangaPageRunner) parseMarkdown(input string) (*domain.MangaResponse, error) {
	manga := &domain.MangaResponse{}
	lines := strings.Split(input, "\n")
	var currentPage *domain.MangaPage

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}
		if m := titleRegex.FindStringSubmatch(trimmedLine); m != nil {
			manga.Title = strings.TrimSpace(m[1])
			continue
		}
		if panelRegex.MatchString(trimmedLine) {
			if currentPage != nil {
				manga.Pages = append(manga.Pages, *currentPage)
			}
			refPath := ""
			if _, after, found := strings.Cut(trimmedLine, ":"); found {
				refPath = strings.TrimSpace(after)
			}
			fullPath := refPath
			if refPath != "" && !strings.HasPrefix(refPath, "http") {
				fullPath = r.baseURL + refPath
			}
			currentPage = &domain.MangaPage{
				Page:         len(manga.Pages) + 1,
				ReferenceURL: fullPath,
			}
			continue
		}
		if currentPage != nil {
			if m := fieldRegex.FindStringSubmatch(trimmedLine); m != nil {
				key, val := strings.ToLower(m[1]), strings.TrimSpace(m[2])
				switch key {
				case "speaker":
					currentPage.SpeakerID = val
				case "text":
					currentPage.Dialogue = val
				case "action":
					currentPage.VisualAnchor = val
				}
			}
		}
	}
	if currentPage != nil {
		manga.Pages = append(manga.Pages, *currentPage)
	}
	return manga, nil
}

// generateMangaPage generates a single unified manga page image based on the provided MangaResponse data.
// It utilizes the configured pipeline to process the manga content and combines layouts, visual anchors, and dialogue.
// Returns the generated ImageResponse or an error if the operation fails.
func (r *MangaPageRunner) generateMangaPage(ctx context.Context, manga domain.MangaResponse) (*imagedom.ImageResponse, error) {
	return r.pipeline.ExecuteMangaPage(ctx, manga)
}
