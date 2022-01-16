package block

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/blockchain/transaction"
	"strings"
	"time"
)

var DUMMY_PARENT_HASH string = strings.Repeat("0", 64)
var GENESIS_DIFFICULTY int = 2
var GENESIS_NONCE uint32 = 0
var GENESIS_BENEFICIARY *account.Address = account.NewAddress([8]byte{})

// DefaultGenesis returns the default genesis block
func DefaultGenesis() *Block {
	bb := NewBlockBuilder(storage.CreateSimpleKV)
	bb.SetParentHash(DUMMY_PARENT_HASH).setDifficulty(GENESIS_DIFFICULTY).
		SetNonce(GENESIS_NONCE).setTimeStamp(0).SetBeneficiary(*GENESIS_BENEFICIARY).SetNumber(0)
	return bb.Build()
}

type BlockHeader struct {
	ParentHash       string // hex form
	Nonce            uint32
	Timestamp        int64 // unix millseconds
	Beneficiary      account.Address
	Difficulty       int
	Number           int
	StateHash        string
	TransactionsHash string
	ReceiptsHash     string
}

func (bh *BlockHeader) hash() []byte {
	var err error = nil
	h := sha256.New()
	writeOnce := func(raw []byte) {
		if err != nil {
			return
		}
		_, err = h.Write(raw)
	}
	writeOnce([]byte(bh.ParentHash))
	writeOnce([]byte(fmt.Sprintf("%d", bh.Nonce)))
	writeOnce([]byte(fmt.Sprintf("%d", bh.Timestamp)))
	writeOnce([]byte(bh.Beneficiary.String()))
	writeOnce([]byte(fmt.Sprintf("%d", bh.Number)))
	writeOnce([]byte(bh.StateHash))
	writeOnce([]byte(bh.TransactionsHash))
	writeOnce([]byte(bh.ReceiptsHash))
	if err != nil {
		panic(err)
	}
	return h.Sum(nil)
}

type Block struct {
	Header BlockHeader
	State  storage.KV // addr -> *addr_state.
	//Transactions storage.KV
	Transactions []*transaction.SignedTransaction
	Receipts     storage.KV
}

type BlockBuilder struct {
	parentHash  string // hex form
	nonce       uint32 // bitcoin style
	timestamp   int64  // unix millseconds
	beneficiary account.Address
	difficulty  int
	number      int
	state       storage.KV // addr -> *account_state
	//transactions storage.KV
	transactions []*transaction.SignedTransaction
	receipts     storage.KV
}

func NewBlockBuilder(factory storage.KVFactory) *BlockBuilder {
	return &BlockBuilder{state: factory(), transactions: make([]*transaction.SignedTransaction, 0), receipts: factory()}
}

func (bb *BlockBuilder) GetDifficulty() int { return bb.difficulty }

func (bb *BlockBuilder) SetWorldState(worldState storage.KV) *BlockBuilder {
	bb.state = worldState
	return bb
}

func (bb *BlockBuilder) SetParentHash(parent string) *BlockBuilder {
	bb.parentHash = parent
	return bb
}

func (bb *BlockBuilder) SetNonce(nonce uint32) *BlockBuilder {
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

func (bb *BlockBuilder) SetDifficulty(difficulty int) *BlockBuilder {
	bb.difficulty = difficulty
	return bb
}

func (bb *BlockBuilder) SetState(state storage.KV) *BlockBuilder {
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

func (bb *BlockBuilder) SetAddrStringState(addr string, s *account.State) *BlockBuilder {
	err := bb.state.Put(addr, s)
	if err != nil {
		panic(err)
	}
	return bb
}

func (bb *BlockBuilder) AddTxn(txn *transaction.SignedTransaction) *BlockBuilder {
	//if err := bb.transactions.Put(hex.EncodeToString(txn.Digest), txn); err != nil {
	//	panic(err)
	//}
	bb.transactions = append(bb.transactions, txn)
	return bb
}

func (bb *BlockBuilder) SetTxns(txns []*transaction.SignedTransaction) *BlockBuilder {
	bb.transactions = txns
	return bb
}

func (bb *BlockBuilder) SetHeader(header BlockHeader) *BlockBuilder {
	bb.parentHash = header.ParentHash
	bb.nonce = header.Nonce
	bb.timestamp = header.Timestamp
	bb.beneficiary = header.Beneficiary
	bb.difficulty = header.Difficulty
	bb.number = header.Number
	return bb
}

func (bb *BlockBuilder) SetReceipts(receipts storage.KV) *BlockBuilder {
	bb.receipts = receipts
	return bb
}

func (bb *BlockBuilder) Build() *Block {
	h := sha256.New()
	for _, txn := range bb.transactions {
		_, err := h.Write(txn.HashBytes())
		if err != nil {
			panic(err)
		}
	}
	header := BlockHeader{
		ParentHash:       bb.parentHash,
		Nonce:            bb.nonce,
		Timestamp:        bb.timestamp,
		Beneficiary:      bb.beneficiary,
		Difficulty:       bb.difficulty,
		Number:           bb.number,
		StateHash:        bb.state.Hash(),
		TransactionsHash: hex.EncodeToString(h.Sum(nil)),
		ReceiptsHash:     bb.receipts.Hash(),
	}
	return &Block{
		Header:       header,
		State:        bb.state,
		Transactions: bb.transactions,
		Receipts:     bb.receipts,
	}

}

func (b *Block) HashBytes() []byte {
	var err error = nil
	h := sha256.New()
	writeOnce := func(raw []byte) {
		if err != nil {
			return
		}
		_, err = h.Write(raw)
	}
	headerHash := b.Header.hash()
	writeOnce(headerHash)
	writeOnce([]byte(b.Header.StateHash))
	writeOnce([]byte(b.Header.TransactionsHash))
	writeOnce([]byte(b.Header.ReceiptsHash))
	if err != nil {
		panic(err)
	}
	return h.Sum(nil)
}

// Hash returns the hex-encoded sha256 bytes
func (b *Block) Hash() string {
	return hex.EncodeToString(b.HashBytes())
}

func (b *Block) NextBlockBuilder(factory storage.KVFactory, miner *account.Address) *BlockBuilder {
	bb := NewBlockBuilder(factory).
		SetParentHash(b.Hash()).
		setDifficulty(b.Header.Difficulty).
		setTimeStamp(time.Now().UnixMilli()).
		SetNumber(b.Header.Number + 1).
		SetBeneficiary(*miner)
	return bb
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

	row1 := fmt.Sprintf("%s| %s", padOrCrop("prev", 6), b.Header.ParentHash)
	row2 := fmt.Sprintf("%s| %d", padOrCrop("idx", 6), b.Header.Number)

	row3 := fmt.Sprintf("%s| %s", padOrCrop("time", 6), time.UnixMilli(b.Header.Timestamp))
	row4 := fmt.Sprintf("%s| %s", padOrCrop("state", 6), b.State)
	row5 := fmt.Sprintf("%s| %s", padOrCrop("txns", 6), b.Transactions)
	row6 := fmt.Sprintf("%s| %s", padOrCrop("recps", 6), b.Receipts)
	row7 := fmt.Sprintf("%s| %d", padOrCrop("nonce", 6), b.Header.Nonce)
	row8 := fmt.Sprintf("%s| %d", padOrCrop("diffi", 6), b.Header.Difficulty)

	maxLen := max(row1, row2, row3, row4, row5, row6, row7, row8)
	row1 = padOrCrop(row1, maxLen)
	row2 = padOrCrop(row2, maxLen)
	row3 = padOrCrop(row3, maxLen)
	row4 = padOrCrop(row4, maxLen)
	row5 = padOrCrop(row5, maxLen)
	row6 = padOrCrop(row6, maxLen)
	row7 = padOrCrop(row7, maxLen)
	row8 = padOrCrop(row8, maxLen)

	ret := ""
	ret += fmt.Sprintf("\n┌%s┐\n", strings.Repeat("─", maxLen+2))
	ret += fmt.Sprintf("|%s  |\n", row1)
	ret += fmt.Sprintf("|%s  |\n", row2)
	ret += fmt.Sprintf("|%s  |\n", row3)
	ret += fmt.Sprintf("|%s  |\n", row4)
	ret += fmt.Sprintf("|%s  |\n", row5)
	ret += fmt.Sprintf("|%s  |\n", row6)
	ret += fmt.Sprintf("|%s  |\n", row7)
	ret += fmt.Sprintf("|%s  |\n", row8)

	ret += fmt.Sprintf("└%s┘\n", strings.Repeat("─", maxLen+2))
	return ret
}
