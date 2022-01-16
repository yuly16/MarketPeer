package miner

import (
	"go.dedis.ch/cs438/blockchain/block"
	"math/rand"
)

// each byte has 8 bits
func satisfyPrefixZeros(hash []byte, zeros int) bool {
	for i := 0; i < zeros; i++ {
		if hash[i] != 0 {
			return false
		}
	}
	return true
}

func difficultyToZeros(difficulty int) int {
	return difficulty
}

func (m *Miner) blockPoW(bb *block.BlockBuilder) *block.Block {
	difficulty := bb.GetDifficulty() // how many zeros
	nonce := rand.Uint32()
	bb.SetNonce(nonce)
	hash := bb.Build().HashBytes()
	for !satisfyPrefixZeros(hash, difficultyToZeros(difficulty)) {
		nonce += 1
		bb.SetNonce(nonce)
		hash = bb.Build().HashBytes()
	}
	return bb.Build()
}
