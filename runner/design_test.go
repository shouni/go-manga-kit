package runner

import (
	"context"
	"io"
	"strings"
	"testing"

	imagePorts "github.com/shouni/gemini-image-kit/ports"
	characterkit "github.com/shouni/go-character-kit/character"
	"github.com/shouni/go-manga-kit/layout"
	"github.com/shouni/go-manga-kit/ports"
	"github.com/shouni/go-remote-io/remoteio"
)

type mockDesignAssetManager struct{}

func (m *mockDesignAssetManager) UploadFile(_ context.Context, refURL string) (string, error) {
	return "https://file-api.google.com/" + refURL, nil
}

func (m *mockDesignAssetManager) DeleteFile(_ context.Context, _ string) error { return nil }

type mockDesignBackend struct{ isVertex bool }

func (m *mockDesignBackend) IsVertexAI() bool { return m.isVertex }

type mockDesignGenerator struct {
	lastReq imagePorts.ImageFusionRequest
}

func (m *mockDesignGenerator) GenerateFusedImage(_ context.Context, req imagePorts.ImageFusionRequest) (*imagePorts.ImageResponse, error) {
	m.lastReq = req
	return &imagePorts.ImageResponse{Data: []byte("fake-png"), MimeType: "image/png", UsedSeed: 1}, nil
}

type mockDesignWriter struct{}

func (m *mockDesignWriter) Write(_ context.Context, _ string, _ io.Reader, _ ...remoteio.WriteOption) error {
	return nil
}

func TestMangaDesignRunner_BuildDesignPromptLayoutKind(t *testing.T) {
	dr := &MangaDesignRunner{}
	descriptions := []string{"Tsumugi (orange hair)"}

	t.Run("default layoutKind uses multi-view turnaround", func(t *testing.T) {
		prompt := dr.buildDesignPrompt(descriptions, "")
		if !strings.Contains(prompt, "multiple views (front, side, back)") {
			t.Errorf("prompt = %q, want multi-view layout text", prompt)
		}
		if strings.Contains(prompt, "single view") {
			t.Errorf("prompt unexpectedly contains single-view layout text: %q", prompt)
		}
	})

	t.Run("DesignLayoutSingleView uses single-pose layout", func(t *testing.T) {
		prompt := dr.buildDesignPrompt(descriptions, DesignLayoutSingleView)
		if !strings.Contains(prompt, "single view, front-facing") {
			t.Errorf("prompt = %q, want single-view layout text", prompt)
		}
		if strings.Contains(prompt, "multiple views") {
			t.Errorf("prompt unexpectedly contains multi-view layout text: %q", prompt)
		}
	})
}

func TestMangaDesignRunner_BuildDesignPromptEmptyDescriptions(t *testing.T) {
	dr := &MangaDesignRunner{}
	if prompt := dr.buildDesignPrompt(nil, ""); prompt != "" {
		t.Errorf("buildDesignPrompt(nil) = %q, want empty string", prompt)
	}
}

func newTestDesignRunner(t *testing.T) (*MangaDesignRunner, *mockDesignGenerator) {
	t.Helper()
	cm, err := characterkit.NewCharacters([]ports.Character{
		{
			ID:           "tsumugi",
			Name:         "Tsumugi",
			ReferenceURL: "gs://bucket/tsumugi.png",
			VisualCues:   []string{"orange hair", "yellow cardigan"},
			IsDefault:    true,
		},
	})
	if err != nil {
		t.Fatalf("NewCharacters failed: %v", err)
	}
	composer, err := layout.NewMangaComposer(&mockDesignAssetManager{}, &mockDesignBackend{isVertex: true}, cm)
	if err != nil {
		t.Fatalf("NewMangaComposer failed: %v", err)
	}
	genMock := &mockDesignGenerator{}
	dr := NewMangaDesignRunner(composer, genMock, &mockDesignWriter{}, "gemini-2.0-flash", "")
	return dr, genMock
}

func TestMangaDesignRunner_RunWithoutOverrideUsesCharacterDefinition(t *testing.T) {
	dr, genMock := newTestDesignRunner(t)

	_, _, err := dr.Run(context.Background(), []string{"tsumugi"}, 42, "gs://bucket/out", "", "", DesignOverride{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(genMock.lastReq.Images) != 1 || genMock.lastReq.Images[0].ReferenceURL != "gs://bucket/tsumugi.png" {
		t.Errorf("Images = %+v, want the character's own ReferenceURL", genMock.lastReq.Images)
	}
	if !strings.Contains(genMock.lastReq.Prompt, "orange hair") {
		t.Errorf("Prompt = %q, want the character's own visual cues", genMock.lastReq.Prompt)
	}
}

func TestMangaDesignRunner_RunSetsSystemAndNegativePrompts(t *testing.T) {
	dr, genMock := newTestDesignRunner(t)

	_, _, err := dr.Run(context.Background(), []string{"tsumugi"}, 42, "gs://bucket/out", "", "", DesignOverride{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if genMock.lastReq.SystemPrompt != designSystemPrompt {
		t.Errorf("SystemPrompt = %q, want designSystemPrompt", genMock.lastReq.SystemPrompt)
	}
	if !strings.Contains(genMock.lastReq.NegativePrompt, "extra fingers") {
		t.Errorf("NegativePrompt = %q, want finger-anatomy negatives", genMock.lastReq.NegativePrompt)
	}
	if !strings.Contains(genMock.lastReq.Prompt, "flat even neutral lighting") {
		t.Errorf("Prompt = %q, want flat lighting constraint appended after styleSuffix", genMock.lastReq.Prompt)
	}
	if !strings.Contains(genMock.lastReq.Prompt, "five fingers per hand") {
		t.Errorf("Prompt = %q, want positive hand-anatomy constraint", genMock.lastReq.Prompt)
	}
}

func TestMangaDesignRunner_RunAppliesOverrideForSingleCharacter(t *testing.T) {
	dr, genMock := newTestDesignRunner(t)

	override := DesignOverride{
		ReferenceURL: "gs://bucket/tsumugi-9x16-draft.png",
		VisualCues:   []string{"temporary test outfit"},
	}
	_, _, err := dr.Run(context.Background(), []string{"tsumugi"}, 42, "gs://bucket/out", "9:16", "", override)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(genMock.lastReq.Images) != 1 || genMock.lastReq.Images[0].ReferenceURL != override.ReferenceURL {
		t.Errorf("Images = %+v, want the overridden ReferenceURL %q", genMock.lastReq.Images, override.ReferenceURL)
	}
	if genMock.lastReq.Images[0].FileAPIURI != "" {
		t.Errorf("FileAPIURI = %q, want empty (override URLs bypass pre-upload)", genMock.lastReq.Images[0].FileAPIURI)
	}
	if !strings.Contains(genMock.lastReq.Prompt, "temporary test outfit") {
		t.Errorf("Prompt = %q, want the overridden visual cues", genMock.lastReq.Prompt)
	}
	if strings.Contains(genMock.lastReq.Prompt, "orange hair") {
		t.Errorf("Prompt = %q, should not contain the character's original visual cues once overridden", genMock.lastReq.Prompt)
	}
}

func TestMangaDesignRunner_RunIgnoresOverrideForMultipleCharacters(t *testing.T) {
	cm, err := characterkit.NewCharacters([]ports.Character{
		{ID: "tsumugi", Name: "Tsumugi", ReferenceURL: "gs://bucket/tsumugi.png", VisualCues: []string{"orange hair"}, IsDefault: true},
		{ID: "metan", Name: "Metan", ReferenceURL: "gs://bucket/metan.png", VisualCues: []string{"purple hair"}},
	})
	if err != nil {
		t.Fatalf("NewCharacters failed: %v", err)
	}
	composer, err := layout.NewMangaComposer(&mockDesignAssetManager{}, &mockDesignBackend{isVertex: true}, cm)
	if err != nil {
		t.Fatalf("NewMangaComposer failed: %v", err)
	}
	genMock := &mockDesignGenerator{}
	dr := NewMangaDesignRunner(composer, genMock, &mockDesignWriter{}, "gemini-2.0-flash", "")

	override := DesignOverride{ReferenceURL: "gs://bucket/should-be-ignored.png"}
	_, _, err = dr.Run(context.Background(), []string{"tsumugi", "metan"}, 42, "gs://bucket/out", "", "", override)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	for _, img := range genMock.lastReq.Images {
		if img.ReferenceURL == override.ReferenceURL {
			t.Errorf("override.ReferenceURL leaked into a multi-character request: %+v", genMock.lastReq.Images)
		}
	}
}
