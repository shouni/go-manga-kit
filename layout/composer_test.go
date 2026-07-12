package layout

import (
	"context"
	"sync/atomic"
	"testing"

	characterkit "github.com/shouni/go-character-kit/character"
	"github.com/shouni/go-manga-kit/ports"
)

// --- Mocks ---

type mockAssetManager struct {
	uploadCount int32
	deleteCount int32
	uploadFunc  func(ctx context.Context, refURL string) (string, error)
}

func (m *mockAssetManager) UploadFile(ctx context.Context, refURL string) (string, error) {
	atomic.AddInt32(&m.uploadCount, 1)
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, refURL)
	}
	return "https://file-api.google.com/" + refURL, nil
}

func (m *mockAssetManager) DeleteFile(_ context.Context, _ string) error {
	atomic.AddInt32(&m.deleteCount, 1)
	return nil
}

type mockBackend struct {
	isVertex bool
}

func (m *mockBackend) IsVertexAI() bool { return m.isVertex }

// --- Tests ---

func TestMangaComposer_PrepareCharacterResources(t *testing.T) {
	ctx := context.Background()
	assetMgr := &mockAssetManager{}
	backend := &mockBackend{isVertex: false}

	cm, err := characterkit.NewCharacters([]ports.Character{
		{
			ID:           "zundamon",
			Name:         "ずんだもん",
			ReferenceURL: "gs://bucket/zunda.png",
			VisualCues:   []string{"green hair"},
		},
		{
			ID:           "metan",
			Name:         "めたん",
			ReferenceURL: "gs://bucket/metan.png",
			VisualCues:   []string{"purple hair"},
			IsDefault:    true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	mc, _ := NewMangaComposer(assetMgr, backend, cm)

	panels := []ports.Panel{
		{SpeakerID: "zundamon"},
		{SpeakerID: "unknown"}, // default (metan) が使用される
	}

	err = mc.PrepareCharacterResources(ctx, panels)
	if err != nil {
		t.Fatalf("PrepareCharacterResources failed: %v", err)
	}

	if uri := mc.GetCharacterResourceURI("zundamon"); uri == "" {
		t.Error("zundamon resource not cached")
	}
	if uri := mc.GetCharacterResourceURI("metan"); uri == "" {
		t.Error("default character (metan) resource not cached")
	}

	if assetMgr.uploadCount != 2 {
		t.Errorf("Expected 2 uploads, got %d", assetMgr.uploadCount)
	}
}

// TestMangaComposer_PrepareCharacterResourcesUploadsAllAspectRatioVariants verifies that
// PrepareCharacterResources uploads both ReferenceURL and every ReferenceURLs entry, and that
// GetCharacterResourceURIFor resolves the aspect-ratio-specific variant when present.
func TestMangaComposer_PrepareCharacterResourcesUploadsAllAspectRatioVariants(t *testing.T) {
	ctx := context.Background()
	assetMgr := &mockAssetManager{}
	backend := &mockBackend{isVertex: false}

	cm, err := characterkit.NewCharacters([]ports.Character{
		{
			ID:           "zundamon",
			Name:         "ずんだもん",
			ReferenceURL: "gs://bucket/zunda-16x9.png",
			ReferenceURLs: map[string]string{
				"1:1": "gs://bucket/zunda-1x1.png",
			},
			VisualCues: []string{"green hair"},
			IsDefault:  true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	mc, _ := NewMangaComposer(assetMgr, backend, cm)

	err = mc.PrepareCharacterResources(ctx, []ports.Panel{{SpeakerID: "zundamon"}})
	if err != nil {
		t.Fatalf("PrepareCharacterResources failed: %v", err)
	}

	if uri := mc.GetCharacterResourceURIFor("zundamon", "1:1"); uri == "" {
		t.Error("1:1 variant resource not cached")
	}
	if uri := mc.GetCharacterResourceURIFor("zundamon", "9:16"); uri == "" {
		t.Error("GetCharacterResourceURIFor should fall back to ReferenceURL when no 9:16 entry exists")
	}

	if assetMgr.uploadCount != 2 {
		t.Errorf("Expected 2 uploads (default + 1:1 variant), got %d", assetMgr.uploadCount)
	}
}
