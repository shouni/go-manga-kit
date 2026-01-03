package director

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// StyleManager は話者の識別や吹き出しの種類（叫び等）を管理します。
type StyleManager struct{}

func NewStyleManager() *StyleManager {
	return &StyleManager{}
}

// ResolveSpeakerID は話者名から CSS 安全なハッシュ ID を生成します。
func (s *StyleManager) ResolveSpeakerID(name string) string {
	if name == "" {
		return "speaker-narration"
	}
	h := sha256.New()
	h.Write([]byte(strings.ToLower(name)))
	return "speaker-" + hex.EncodeToString(h.Sum(nil))[:10]
}

// DetermineDialogueType はセリフに含まれるメタタグから吹き出しの種類を判定します。
func (s *StyleManager) DetermineDialogueType(text string) string {
	switch {
	case strings.Contains(text, "[shout]"):
		return "shout"
	case strings.Contains(text, "[thought]"):
		return "thought"
	default:
		return "normal"
	}
}
