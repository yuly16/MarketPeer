package block

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go.dedis.ch/cs438/blockchain/transaction"
	"sync"

	"go.dedis.ch/cs438/blockchain/storage"
)

var ErrBlockTooNew = errors.New("block is too new")

// BlockChain is a chain of Blocks
type BlockChain struct {
	// TODO: may be we could move this mu to miner level, then it will be easier to synchronize
	mu        sync.Mutex
	blocksMap map[string]*Block
	ends      []*Block // ends has same parent hash, they are forks in the end. their number is the same
}

func NewBlockChain() *BlockChain {
	genesis := DefaultGenesis()
	return NewBlockChainWithGenesis(genesis)
}

func NewBlockChainWithGenesis(genesis *Block) *BlockChain {
	return &BlockChain{
		blocksMap: map[string]*Block{genesis.Hash(): genesis}, ends: []*Block{genesis}}
}

func (bc *BlockChain) GetBlock(number int) (*Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	end := bc.ends[0]
	if number < 0 {
		return nil, fmt.Errorf("number(%d)<0", number)
	}
	if end.Header.Number < number {
		return nil, fmt.Errorf("number(%d) too new, end's number=%d", number, end.Header.Number)
	}
	if end.Header.Number == number {
		return end, nil
	}
	ptr := end.Header.ParentHash
	for ptr != DUMMY_PARENT_HASH {
		b := bc.blocksMap[ptr]
		if b.Header.Number == number {
			return b, nil
		}
		ptr = b.Header.ParentHash
	}
	panic(fmt.Errorf("it shall not be reached"))
}

func (bc *BlockChain) OverWrite(blocks []*Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	genesis := blocks[len(blocks)-1]
	bc.blocksMap = map[string]*Block{genesis.Hash(): genesis}
	bc.ends = []*Block{genesis}
	for i := len(blocks) - 2; i >= 0; i-- {
		if err := bc.Append(blocks[i]); err != nil {
			return err
		}
	}
	return nil
}

func (bc *BlockChain) Len() int {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	ret := 1
	ptr := bc.ends[0].Header.ParentHash
	for ptr != DUMMY_PARENT_HASH {
		ret += 1
		ptr = bc.blocksMap[ptr].Header.ParentHash
	}
	return ret
}

// LatestWorldState returns a copy of the world state stored in the last block
// TODO: do we need a lock?
func (bc *BlockChain) LatestWorldState() (storage.KV, *Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if len(bc.ends) == 1 {
		return bc.ends[0].State.Copy(), bc.ends[0], nil
	} else {
		return nil, nil, fmt.Errorf("ends not decided yet")
	}
}

// LastBlock find ends block with hash
//func (bc *BlockChain) LastBlock(hash string) (*Block, error) {
//	bc.mu.Lock()
//	defer bc.mu.Unlock()
//	for _, b := range bc.ends {
//		if b.Hash() == hash {
//			return b, nil
//		}
//	}
//	return nil, fmt.Errorf("ends has no block's hash=%s", hash)
//}

//func (bc *BlockChain) LastBlock() *Block {
//	bc.mu.Lock()
//	defer bc.mu.Unlock()
//	return bc.blocks[len(bc.blocks)-1]
//}

// hasTxn returns 1. has or not 2. #blocks After
func (bc *BlockChain) HasTxn(handle transaction.SignedTransactionHandle) (bool, int) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	nBlocksAfter := 0
	for _, end := range bc.ends {
		if end.HasTxn(handle) {
			return true, 0
		}
	}
	validEnd := bc.ends[0]
	ptr := validEnd.Header.ParentHash
	for ptr != DUMMY_PARENT_HASH {
		nBlocksAfter += 1
		b := bc.blocksMap[ptr]
		if b.HasTxn(handle) {
			return true, nBlocksAfter
		}
		ptr = b.Header.ParentHash
	}
	return false, -1
}

// TryAppend test if we could append, if could, return the parent for replay the txns
func (bc *BlockChain) TryAppend(block *Block) (*Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	endsNumber := bc.ends[0].Header.Number
	// too new
	if block.Header.Number-endsNumber >= 2 {
		return nil, fmt.Errorf("block too new(number=%d), cannot connect to ends(number=%d), block=%s; %w",
			block.Header.Number, endsNumber, block.String(), ErrBlockTooNew)
	}

	if block.Header.Number-endsNumber <= 0 {
		return nil, fmt.Errorf("block number(%d) not valid, cannot connect to ends(number=%d), block=%s",
			block.Header.Number, endsNumber, block.String())
	}

	// TODO: now we only allow len(ends) = 0
	//// block either connect with ends or connect with ends' parent hash
	//endParentHash := bc.ends[0].Header.ParentHash
	//// block connect with prevEnds, then it also becomes an end
	//if block.Header.ParentHash == endParentHash {
	//	// double cross-check
	//	if block.Header.Number != endsNumber {
	//		panic(fmt.Errorf("fatal error, block numbering is confilcted with parentHash"))
	//	}
	//	return bc.blocksMap[endParentHash], nil
	//}

	for _, b := range bc.ends {
		// block connect with ends, then ends is flushed, and block will become the only end
		if b.Hash() == block.Header.ParentHash {
			if block.Header.Number != endsNumber+1 {
				panic(fmt.Errorf("fatal error, block numbering is confilcted with parentHash"))
			}
			return b, nil
		}
	}
	return nil, fmt.Errorf("block(parentHash=%s) cannot be connected to ends(%s), block=%s",
		block.Hash()[:6]+"...",
		bc.ends[0].Hash()[:6]+"...", block.String())
}

func (bc *BlockChain) Append(block *Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	endsNumber := bc.ends[0].Header.Number
	// too new
	if block.Header.Number-endsNumber >= 2 {
		return fmt.Errorf("block too new(number=%d), cannot connect to ends(number=%d), block=%s; %w",
			block.Header.Number, endsNumber, block.String(), ErrBlockTooNew)
	}

	// TODO: now we only allow len(ends) = 0
	// block either connect with ends or connect with ends' parent hash
	//endParentHash := bc.ends[0].Header.ParentHash
	//// block connect with prevEnds, then it also becomes an end
	//if block.Header.ParentHash == endParentHash {
	//	// double cross-check
	//	if block.Header.Number != endsNumber {
	//		panic(fmt.Errorf("fatal error, block numbering is confilcted with parentHash"))
	//	}
	//
	//	bc.blocksMap[block.Hash()] = block
	//	bc.ends = append(bc.ends, block)
	//	return nil
	//}

	for _, b := range bc.ends {
		// block connect with ends, then ends is flushed, and block will become the only end
		if b.Hash() == block.Header.ParentHash {
			if block.Header.Number != endsNumber+1 {
				panic(fmt.Errorf("fatal error, block numbering is confilcted with parentHash"))
			}
			// flush the ends
			bc.ends = bc.ends[:0]
			// new ends with only one component
			bc.ends = append(bc.ends, block)
			bc.blocksMap[block.Hash()] = block
			return nil
		}
	}
	return fmt.Errorf("block too old, cannot connect to ends, block=%s", block.String())
}

func (bc *BlockChain) HashBytes() []byte {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	h := sha256.New()
	canicalEnd := bc.ends[0]
	for _, end := range bc.ends {
		h.Write(end.HashBytes())
	}
	prevEndsHash := canicalEnd.Header.ParentHash
	ptr := prevEndsHash
	for ptr != DUMMY_PARENT_HASH {
		b := bc.blocksMap[ptr]
		h.Write(b.HashBytes())
		ptr = b.Header.ParentHash
	}
	return h.Sum(nil)
}

func (bc *BlockChain) Hash() string {
	return hex.EncodeToString(bc.HashBytes())
}

func (bc *BlockChain) String() string {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if len(bc.blocksMap) == 0 {
		return ""
	}

	// from latest to oldest
	arrow := "↑\n|\n"
	ret := ""
	// first print ends
	ret += "ends:\n"
	for _, end := range bc.ends {
		ret += end.String()
	}
	prevEndsHash := bc.ends[0].Header.ParentHash
	ptr := prevEndsHash
	for ptr != DUMMY_PARENT_HASH {
		b := bc.blocksMap[ptr]
		ret += arrow
		ret += b.String()
		ptr = b.Header.ParentHash
	}

	//ret += bc.blocks[len(bc.blocks)-1].String()
	//for i := len(bc.blocks) - 2; i >= 0; i-- {
	//	b := bc.blocks[i]
	//	ret += arrow
	//	ret += b.String()
	//}
	return ret
}
