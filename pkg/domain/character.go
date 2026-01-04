package domain

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
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

// CharactersMap はIDや名前をキーとしたキャラクターの検索用マップなのだ。
type CharactersMap map[string]Character

var (
	cachedChars map[string]Character
	once        sync.Once
	loadErr     error
)

// LoadCharacters は指定されたファイルパスからJSONを読み込み、キャラクターマップを返すのだ。
func LoadCharacters(path string) (map[string]Character, error) {
	// 1. ファイルの読み込み
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("キャラクターファイルの読み込みに失敗したのだ: %w", err)
	}

	// 2. バイト列からのパース処理（getCharacters）を再利用するのだ
	return getCharacters(data)
}

// getCharacters はJSONバイト列からキャラクターマップをパースして返すのだ。
func getCharacters(charactersJSON []byte) (map[string]Character, error) {
	// シングルトンでの読み込み（cachedCharsへの格納）
	once.Do(func() {
		if err := json.Unmarshal(charactersJSON, &cachedChars); err != nil {
			loadErr = fmt.Errorf("キャラクター設定のデコードに失敗したのだ: %w", err)
		}
	})

	if loadErr != nil {
		return nil, loadErr
	}

	// 内部キャッシュが呼び出し元によって変更されるのを防ぐため、ディープコピーを返すのだ。
	return copyCharactersMap(cachedChars), nil
}

// copyCharactersMap はマップの防御的コピーを行う内部ヘルパーなのだ。
func copyCharactersMap(src map[string]Character) map[string]Character {
	copied := make(map[string]Character, len(src))
	for k, v := range src {
		charCopy := v
		// VisualCuesスライスも新しく割り当ててコピーするのだ
		if v.VisualCues != nil {
			charCopy.VisualCues = make([]string, len(v.VisualCues))
			copy(charCopy.VisualCues, v.VisualCues)
		}
		copied[k] = charCopy
	}
	return copied
}

// String はキャラクターの情報を文字列で返すのだ。
func (c Character) String() string {
	return fmt.Sprintf("%s (%s)", c.Name, c.ID)
}

// BuildCharactersMap はスライス形式のデータを検索効率の良いマップ形式に変換するのだ。
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

// NewCharacter は名前と特徴からキャラクター構造体を生成します。
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
