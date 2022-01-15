package block

import (
	"sync"

	"go.dedis.ch/cs438/blockchain/storage"
)

// BlockChain is a chain of Blocks
type BlockChain struct {
	mu          sync.Mutex
	blocks      []*Block   // let's first store it in an array
	latestState storage.KV // it is a copy of the latest state
}

func NewBlockChain() *BlockChain {
	genesis := DefaultGenesis()
	return NewBlockChainWithGenesis(genesis)
}

func NewBlockChainWithGenesis(genesis *Block) *BlockChain {
	return &BlockChain{blocks: []*Block{genesis}, latestState: genesis.State.Copy()}
}

// LatestWorldState returns a copy of the world state stored in the last block
// TODO: do we need a lock?
func (bc *BlockChain) LatestWorldState() (storage.KV, *Block) {
	bc.mu.Lock()
	bc.mu.Unlock()
	return bc.latestState, bc.blocks[len(bc.blocks)-1]
}

func (bc *BlockChain) LastBlock() *Block {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	return bc.blocks[len(bc.blocks)-1]
}

func (bc *BlockChain) Append(block *Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.blocks = append(bc.blocks, block)
	bc.latestState = block.State.Copy()
	return nil
}

func (bc *BlockChain) String() string {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	// from latest to oldest
	arrow := "↑\n|\n"
	ret := ""
	if len(bc.blocks) == 0 {
		return ""
	}
	ret += bc.blocks[len(bc.blocks)-1].String()
	for i := len(bc.blocks) - 2; i >= 0; i-- {
		b := bc.blocks[i]
		ret += arrow
		ret += b.String()
	}
	return ret
}
