package runner

import (
	"strings"
	"testing"
)

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
