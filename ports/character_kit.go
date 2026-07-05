// Package ports は、漫画生成ワークフローの外部境界となるインターフェースと
// データ構造を定義します。
package ports

import characterkit "github.com/shouni/go-character-kit/character"

// Character は go-character-kit のキャラクター型のエイリアスです。
type Character = characterkit.Character

// Characters は go-character-kit のキャラクター集合型のエイリアスです。
type Characters = characterkit.Characters
