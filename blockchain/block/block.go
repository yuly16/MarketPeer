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

var DUMMY_PARENT_HASH string = strings.Repeat("0", 64)

type BlockHeader struct {
	parentHash       string // hex form
	nonce            string // TODO
	timestamp        int64  // unix millseconds
	beneficiary      account.Address
	difficulty       int
	number           int
	stateHash        string
	transactionsHash string
	receiptsHash     string
}

type Block struct {
	header       BlockHeader
	state        storage.KV // addr -> addr_state. addr_state is json_marshaled
	transactions storage.KV
	receipts     storage.KV
}

type BlockBuilder struct {
	parentHash   string // hex form
	nonce        string // TODO
	timestamp    int64  // unix millseconds
	beneficiary  account.Address
	difficulty   int
	number       int
	state        storage.KV // addr -> *account_state
	transactions storage.KV
	receipts     storage.KV
}

func NewBlockBuilder(factory storage.KVFactory) *BlockBuilder {
	return &BlockBuilder{state: factory(), transactions: factory(), receipts: factory()}
}

func (bb *BlockBuilder) SetParentHash(parent string) *BlockBuilder {
	bb.parentHash = parent
	return bb
}

func (bb *BlockBuilder) SetNonce(nonce string) *BlockBuilder {
	bb.nonce = nonce
	return bb
}

func (bb *BlockBuilder) setTimeStamp(stamp int64) *BlockBuilder {
	bb.timestamp = stamp
	return bb
}

func (bb *BlockBuilder) SetBeneficiary(beneficiary account.Address) *BlockBuilder {
	bb.beneficiary = beneficiary
	return bb
}

func (bb *BlockBuilder) setDifficulty(difficulty int) *BlockBuilder {
	bb.difficulty = difficulty
	return bb
}

func (bb *BlockBuilder) SetNumber(number int) *BlockBuilder {
	bb.number = number
	return bb
}

func (bb *BlockBuilder) setState(state storage.KV) *BlockBuilder {
	bb.state = state
	return bb
}

func (bb *BlockBuilder) SetAddrState(addr *account.Address, s *account.State) *BlockBuilder {
	err := bb.state.Put(addr.String(), s)
	if err != nil {
		panic(err)
	}
	return bb
}

func (bb *BlockBuilder) setTxns(txns storage.KV) *BlockBuilder {
	bb.transactions = txns
	return bb
}

func (bb *BlockBuilder) setReceipts(receipts storage.KV) *BlockBuilder {
	bb.receipts = receipts
	return bb
}

func (bb *BlockBuilder) Build() *Block {
	header := BlockHeader{
		parentHash:       bb.parentHash,
		nonce:            bb.nonce,
		timestamp:        bb.timestamp,
		beneficiary:      bb.beneficiary,
		difficulty:       bb.difficulty,
		number:           bb.number,
		stateHash:        bb.state.Hash(),
		transactionsHash: bb.transactions.Hash(),
		receiptsHash:     bb.receipts.Hash(),
	}
	return &Block{
		header:       header,
		state:        bb.state,
		transactions: bb.transactions,
		receipts:     bb.receipts,
	}

}

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

	padOrCrop := func(s string, maxlen int) string {
		if len(s) >= maxlen {
			return s[:maxlen]
		} else {
			return s + strings.Repeat(" ", maxlen-len(s))
		}
	}

	row1 := fmt.Sprintf("%s| %s", padOrCrop("prev", 6), b.header.parentHash)
	row2 := fmt.Sprintf("%s| %d", padOrCrop("idx", 6), b.header.number)
	row3 := fmt.Sprintf("%s| %s", padOrCrop("time", 6), time.UnixMilli(b.header.timestamp))
	row4 := fmt.Sprintf("%s| %s", padOrCrop("state", 6), b.state)
	row5 := fmt.Sprintf("%s| %s", padOrCrop("txns", 6), b.transactions)
	row6 := fmt.Sprintf("%s| %s", padOrCrop("recps", 6), b.receipts)

	maxLen := max(row1, row2, row3, row4, row5, row6)
	row1 = padOrCrop(row1, maxLen)
	row2 = padOrCrop(row2, maxLen)
	row3 = padOrCrop(row3, maxLen)
	row4 = padOrCrop(row4, maxLen)
	row5 = padOrCrop(row5, maxLen)
	row6 = padOrCrop(row6, maxLen)

	ret := ""
	ret += fmt.Sprintf("\n┌%s┐\n", strings.Repeat("─", maxLen+2))
	ret += fmt.Sprintf("|%s  |\n", row1)
	ret += fmt.Sprintf("|%s  |\n", row2)
	ret += fmt.Sprintf("|%s  |\n", row3)
	ret += fmt.Sprintf("|%s  |\n", row4)
	ret += fmt.Sprintf("|%s  |\n", row5)
	ret += fmt.Sprintf("|%s  |\n", row6)

	ret += fmt.Sprintf("└%s┘\n", strings.Repeat("─", maxLen+2))
	return ret
}
