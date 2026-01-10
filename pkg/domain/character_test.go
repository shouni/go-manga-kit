package domain

import (
	"testing"
)

func TestGetCharacters(t *testing.T) {
	// 1. 正常系：正しいJSONからマップが生成されること
	jsonInput := []byte(`{
		"hero": {
			"id": "hero",
			"name": "勇者",
			"visual_cues": ["blue hair", "sword"],
			"seed": 123,
			"is_primary": true
		}
	}`)

	chars, err := GetCharacters(jsonInput)
	if err != nil {
		t.Fatalf("正常なJSONでエラーが発生しました: %v", err)
	}

	if chars["hero"].Name != "勇者" {
		t.Errorf("期待値 '勇者', 実際の値 '%s'", chars["hero"].Name)
	}

	// 2. 異常系：不正なJSONでエラーが返ること
	_, err = GetCharacters([]byte(`{ invalid json }`))
	if err == nil {
		t.Error("不正なJSONでエラーが発生しませんでした")
	}
}

func TestGetSeedFromName(t *testing.T) {
	// テスト用のデータ準備
	chars := CharactersMap{
		"Alice": Character{ID: "alice", Name: "Alice", Seed: 999},
		"Bob":   Character{ID: "bob", Name: "Bob", Seed: 0}, // Seed未設定
	}

	t.Run("設定済みのSeedを取得できること", func(t *testing.T) {
		seed := GetSeedFromName("Alice", chars)
		if seed != 999 {
			t.Errorf("期待値 999, 実際の値 %d", seed)
		}
	})

	t.Run("Seed未設定の場合はハッシュから生成されること", func(t *testing.T) {
		seed := GetSeedFromName("Bob", chars)
		if seed == 0 {
			t.Error("Seedが0のままです。ハッシュ生成が行われていない可能性があります")
		}
	})

	t.Run("マップに存在しない名前でも決定論的にSeedが生成されること", func(t *testing.T) {
		seed1 := GetSeedFromName("Unknown", chars)
		seed2 := GetSeedFromName("Unknown", chars)

		if seed1 == 0 {
			t.Error("Seedが0です")
		}
		if seed1 != seed2 {
			t.Error("同じ名前から異なるSeedが生成されました。決定論的ではありません")
		}
	})
}

func TestCharacter_String(t *testing.T) {
	c := Character{ID: "test-id", Name: "テスト名"}
	expected := "テスト名 (test-id)"
	if c.String() != expected {
		t.Errorf("期待値 '%s', 実際の値 '%s'", expected, c.String())
	}
}
