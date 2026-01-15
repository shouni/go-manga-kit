package domain

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// FindCharacter は 直接のIDからキャラクター情報を特定します。
func (m CharactersMap) FindCharacter(ID string) *Character {
	if m == nil {
		return nil
	}
	if char, ok := m[ID]; ok {
		res := char
		return &res
	}
	if char, ok := m[strings.ToLower(ID)]; ok {
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
