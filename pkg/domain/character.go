package domain

import (
	"encoding/json"
	"fmt"
)

// Character は漫画に登場するキャラクターの定義を保持します。
type Character struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	VisualCues   []string `json:"visual_cues"`   // 生成プロンプトに注入する外見上の特徴
	ReferenceURL string   `json:"reference_url"` // 一貫性保持のための参照画像URL
	Seed         int64    `json:"seed"`          // DB保存等のために広い型を維持
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

// String はキャラクターの情報を文字列で返します。
func (c Character) String() string {
	return fmt.Sprintf("%s (%s)", c.Name, c.ID)
}
