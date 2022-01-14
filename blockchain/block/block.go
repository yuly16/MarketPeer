package block

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/storage"
	"strings"
	"time"
)

type BlockHeader struct {
	parentHash       string // hex form
	nonce            string // TODO
	timestamp        int64  // unix millseconds
	beneficiary      account.Address
	difficulty       int
	number           uint32
	stateHash        string
	transactionsHash string
	receiptsHash     string
}

type Block struct {
	header       BlockHeader
	state        storage.KV
	transactions storage.KV
	receipts     storage.KV
}

type BlockBuilder struct {
	parentHash string // hex form
	nonce      string // TODO

}

func (bb *BlockBuilder) setParentHash(parent string) *BlockBuilder {
	bb.parentHash = parent
	return bb
}

func (bb *BlockBuilder) setNonce(nonce string) *BlockBuilder {
	bb.nonce = nonce
	return bb
}

//func (bb *BlockBuilder) build() *Block {
//
//}

// Hash returns the hex-encoded sha256 bytes
func (b *Block) Hash() string {
	raw := []byte(fmt.Sprintf(""))
	h := sha256.New()
	_, err := h.Write(raw)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (b *Block) String() string {
	max := func(s ...string) int {
		max := 0
		for _, se := range s {
			if len(se) > max {
				max = len(se)
			}
		}
		return max
	}

	row1 := fmt.Sprintf("prev | %s", b.header.parentHash)
	row2 := fmt.Sprintf("idx | %d", b.header.number)
	row3 := fmt.Sprintf("time | %s", time.UnixMilli(b.header.timestamp))
	maxLen := max(row1, row2, row3)

	ret := ""
	ret += fmt.Sprintf("\n┌%s┐\n", strings.Repeat("─", maxLen+2))
	ret += fmt.Sprintf("|%s|\n", row1)
	ret += fmt.Sprintf("|%s|\n", row2)
	ret += fmt.Sprintf("|%s|\n", row3)
	ret += fmt.Sprintf("└%s┘\n", strings.Repeat("─", maxLen+2))
	return ret
}
