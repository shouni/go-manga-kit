package generator

import (
	"crypto/sha256"
	"encoding/binary"
)

// CharacterDNA はキャラクターの視覚的一貫性を保つための情報を保持します。
type CharacterDNA struct {
	Name      string // キャラクター識別子
	VisualCue string // 外見上の特徴（例: "green hair, ribbon, school uniform"）
	Seed      int32  // 生成時の一貫性を保つためのシード値
}

// DNAMap はキャラクター名とDNAの対応表を管理します。
type DNAMap map[string]CharacterDNA

// GetSeedFromName は名前から決定論的なシード値を生成します。
// これにより、明示的なシード指定がない場合でも名前が同じなら同じシードが使われます。
func GetSeedFromName(name string) int32 {
	hash := sha256.Sum256([]byte(name))
	// ハッシュの最初の4バイトを int32 に変換
	seed := int32(binary.BigEndian.Uint32(hash[:4]))
	// Geminiのシード値は正の数が望ましいため、絶対値をとるかビットマスクします
	if seed < 0 {
		seed = -seed
	}
	return seed
}

// NewCharacterDNA は名前と特徴からDNA構造体を生成します。
func NewCharacterDNA(name, visualCue string, seed int32) CharacterDNA {
	if seed == 0 {
		seed = GetSeedFromName(name)
	}
	return CharacterDNA{
		Name:      name,
		VisualCue: visualCue,
		Seed:      seed,
	}
}
