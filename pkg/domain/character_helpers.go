package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// FindCharacter は 直接のID一致、ハッシュIDからキャラクター情報を特定します。
func (m CharactersMap) FindCharacter(ID string) *Character {
	if m == nil {
		return nil
	}

	// 1. 完全一致（最優先）
	if char, ok := m[ID]; ok {
		res := char
		return &res
	}

	sid := strings.ToLower(ID)

	h := sha256.New()
	for _, char := range m {
		h.Reset()
		h.Write([]byte(char.ID))
		hash := hex.EncodeToString(h.Sum(nil))
		if sid == "speaker-"+hash[:10] {
			res := char
			return &res
		}
	}

	// 3. 接頭辞 "speaker-" を除去して再試行
	cleanID := strings.TrimPrefix(sid, "speaker-")
	if char, ok := m[cleanID]; ok {
		res := char
		return &res
	}

	return nil
}

// GetPrimary はマップ内から IsPrimary が true のキャラクターを1人返します。
// 常に同じ結果を得るため、IDでソートした順に走査します。
func (m CharactersMap) GetPrimary() *Character {
	if len(m) == 0 {
		return nil
	}

	// 1. キー（ID）を抽出してソート
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. ソート順に走査して最初に見つかった Primary を返す
	for _, k := range keys {
		char := m[k]
		if char.IsPrimary {
			res := char
			return &res
		}
	}

	return nil
}

// GetCharacters はJSONバイト列からキャラクターマップをパースして返します。
// この関数はステートレスであり、キャッシュを行いません。
func GetCharacters(charactersJSON []byte) (CharactersMap, error) {
	var chars CharactersMap
	if err := json.Unmarshal(charactersJSON, &chars); err != nil {
		return nil, fmt.Errorf("キャラクター情報のJSONパースに失敗しました: %w", err)
	}
	return chars, nil
}
