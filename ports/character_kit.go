package ports

import characterkit "github.com/shouni/go-character-kit/character"

type Character = characterkit.Character
type Characters = characterkit.Characters

// NewCharacters はキャラクター定義を検証し、ID検索用インデックスを構築します。
func NewCharacters(list []Character) (*Characters, error) {
	return characterkit.NewCharacters(list)
}

func ParseCharacters(charactersJSON []byte) (*Characters, error) {
	return characterkit.ParseCharacters(charactersJSON)
}
