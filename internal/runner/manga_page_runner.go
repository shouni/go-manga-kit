package runner

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/shouni/go-manga-kit/pkg/domain"
	mangaPipeline "github.com/shouni/go-manga-kit/pkg/pipeline"

	imagedom "github.com/shouni/gemini-image-kit/pkg/domain"
)

var (
	titleRegex = regexp.MustCompile(`^#\s+(.+)`)
	panelRegex = regexp.MustCompile(`^##\s+Panel:?`)
	fieldRegex = regexp.MustCompile(`^\s*-\s*([a-zA-Z_]+):\s*(.+)`)
)

// MangaPageRunner は MarkdownのパースとPipelineの実行を管理するのだ。
type MangaPageRunner struct {
	pipeline *mangaPipeline.PagePipeline
	baseURL  string
}

func NewMangaPageRunner(manga mangaPipeline.Pipeline, styleSuffix string, scriptURL string) *MangaPageRunner {
	baseURL := ""
	u, err := url.Parse(scriptURL)
	if err == nil && u.Scheme == "gs" {
		baseURL = fmt.Sprintf("https://storage.googleapis.com/%s%s/", u.Host, path.Dir(u.Path))
	}

	return &MangaPageRunner{
		pipeline: mangaPipeline.NewPagePipeline(manga, styleSuffix), // manga全体を渡す
		baseURL:  baseURL,
	}
}

func (r *MangaPageRunner) Run(ctx context.Context, manga domain.MangaResponse) (*imagedom.ImageResponse, error) {
	return r.pipeline.ExecuteMangaPage(ctx, manga)
}

func (r *MangaPageRunner) RunMarkdown(ctx context.Context, markdownContent string) (*imagedom.ImageResponse, error) {
	manga, err := r.ParseMarkdown(markdownContent)
	if err != nil {
		return nil, fmt.Errorf("Markdownのパースに失敗したのだ: %w", err)
	}
	return r.Run(ctx, *manga)
}

func (r *MangaPageRunner) ParseMarkdown(input string) (*domain.MangaResponse, error) {
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
