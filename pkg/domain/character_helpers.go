package domain

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

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
func GetCharacters(charactersJSON []byte) (map[string]Character, error) {
	var chars map[string]Character
	if err := json.Unmarshal(charactersJSON, &chars); err != nil {
		return nil, fmt.Errorf("キャラクター情報のJSONパースに失敗しました: %w", err)
	}
	return chars, nil
}

// GetSeedFromName は名前から決定論的なシード値を生成、または設定値から取得します。
func GetSeedFromName(name string, chars CharactersMap) int64 {
	if c, ok := chars[name]; ok && c.Seed != 0 {
		return c.Seed
	}
	// 設定がない場合は名前からハッシュを生成してシードにするのだ
	hash := sha256.Sum256([]byte(name))
	return int64(binary.BigEndian.Uint64(hash[:8]))
}

// FindCharacter は ID からキャラクター情報を特定します。
func FindCharacter(ID string, characters map[string]Character) *Character {
	sid := strings.ToLower(ID)
	h := sha256.New()
	for _, char := range characters {
		h.Reset()
		h.Write([]byte(char.ID))
		hash := hex.EncodeToString(h.Sum(nil))
		if sid == "speaker-"+hash[:10] {
			return &char
		}
	}
	cleanID := strings.TrimPrefix(sid, "speaker-")
	if char, ok := characters[cleanID]; ok {
		return &char
	}
	return nil
}
