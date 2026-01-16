package domain

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// FindCharacter は、指定されたID（またはその小文字版）からキャラクター情報を特定します。
// マップに存在する場合はそのポインタを返し、存在しない場合は nil を返します。
func (m CharactersMap) FindCharacter(ID string) *Character {
	if m == nil {
		return nil
	}

	// 1. 直接のIDで検索、見つからなければ小文字に正規化して再検索
	char, ok := m[ID]
	if !ok {
		char, ok = m[strings.ToLower(ID)]
	}

	if ok {
		// マップから取得した値（コピー）のアドレスを直接返します
		return &char
	}

	return nil
}

// GetPrimary はマップ内から IsPrimary が true のキャラクターを1人返します。
// 常に決定論的な結果を得るため、IDでソートした順に走査します。
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
			return &char
		}
	}

	return nil
}

// GetCharacters はJSONバイト列からキャラクターマップをパースして返します。
func GetCharacters(charactersJSON []byte) (CharactersMap, error) {
	var chars CharactersMap
	if err := json.Unmarshal(charactersJSON, &chars); err != nil {
		return nil, fmt.Errorf("キャラクター情報のJSONパースに失敗しました: %w", err)
	}
	return chars, nil
}

// GetSeedFromString は文字列から決定論的なシード値を生成します。
func GetSeedFromString(s string) int64 {
	if s == "" {
		return 0
	}
	hash := sha256.Sum256([]byte(s))
	return int64(binary.BigEndian.Uint64(hash[:8]))
}
