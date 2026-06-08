package ports

import (
	"strings"

	characterkit "github.com/shouni/go-character-kit/character"
)

type Character = characterkit.Character
type Characters = characterkit.Characters

// NewCharacters はキャラクター定義を検証し、ID検索用インデックスを構築します。
func NewCharacters(list []Character) (*Characters, error) {
	chars := &Characters{
		List: list,
		ByID: make(map[string]*Character, len(list)*2),
	}
	if err := chars.Validate(); err != nil {
		return nil, err
	}

	for i := range chars.List {
		char := &chars.List[i]
		chars.ByID[char.ID] = char
		chars.ByID[strings.ToLower(char.ID)] = char
	}
	return chars, nil
}

func ParseCharacters(charactersJSON []byte) (*Characters, error) {
	return characterkit.ParseCharacters(charactersJSON)
}
