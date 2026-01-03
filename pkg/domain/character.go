package domain

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"
)

// Character は漫画に登場するキャラクターの定義を保持します。
type Character struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	VisualCues   []string `json:"visual_cues"`   // 生成プロンプトに注入する外見上の特徴
	ReferenceURL string   `json:"reference_url"` // 一貫性保持のための参照画像URL
	Seed         int64    `json:"seed"`          // DB保存等のために広い型を維持
}

type CharactersMap map[string]Character

var (
	cachedChars map[string]Character
	once        sync.Once
	loadErr     error
)

// GetCharacters は埋め込まれたJSONからキャラクターマップを返すのだ。
func GetCharacters(charactersJSON []byte) (map[string]Character, error) {
	once.Do(func() {
		if err := json.Unmarshal(charactersJSON, &cachedChars); err != nil {
			loadErr = fmt.Errorf("キャラクター設定の読み込みに失敗したのだ: %w", err)
		}
	})

	if loadErr != nil {
		return nil, loadErr
	}

	// 内部キャッシュが呼び出し元によって変更されるのを防ぐため、マップの防御的コピーを返します。
	copiedChars := make(map[string]Character, len(cachedChars))
	for k, v := range cachedChars {
		charCopy := v
		if v.VisualCues != nil {
			charCopy.VisualCues = make([]string, len(v.VisualCues))
			copy(charCopy.VisualCues, v.VisualCues)
		}
		copiedChars[k] = charCopy
	}

	return copiedChars, nil
}

func (c Character) String() string {
	return fmt.Sprintf("%s (%s)", c.Name, c.ID)
}

// BuildCharactersMap はスライス形式の DNA データを検索効率の良いマップ形式に変換するのだ
func BuildCharactersMap(chars []Character) CharactersMap {
	m := make(CharactersMap)
	for _, c := range chars {
		key := c.ID
		if key == "" {
			key = c.Name
		}
		m[key] = c
	}
	return m
}

// GetSeedFromName は名前から決定論的なシード値を生成します。
func GetSeedFromName(name string) int32 {
	hash := sha256.Sum256([]byte(name))
	// ハッシュの最初の4バイトを int32 に変換
	seed := int32(binary.BigEndian.Uint32(hash[:4]))
	// Geminiのシード値は正の数が望ましいため、最上位ビットを落とすのが安全なのだ
	return seed & 0x7FFFFFFF
}

// NewCharacter は名前と特徴からDNA構造体を生成します。
func NewCharacter(id, name, visualCue string, seed int32) Character {
	if seed == 0 {
		seed = GetSeedFromName(name)
	}
	// VisualCues はスライスなので、[]string で初期化して代入するのだ。
	return Character{
		ID:         id,
		Name:       name,
		VisualCues: []string{visualCue},
		Seed:       int64(seed),
	}
}
