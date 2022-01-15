package block

import "go.dedis.ch/cs438/blockchain/storage"

// BlockChain is a chain of Blocks
type BlockChain struct {
	blocks      []*Block   // let's first store it in an array
	latestState storage.KV // it is a copy of the latest state
}

func NewBlockChain() *BlockChain {
	genesis := DefaultGenesis()
	return NewBlockChainWithGenesis(genesis)
}

func NewBlockChainWithGenesis(genesis *Block) *BlockChain {
	return &BlockChain{blocks: []*Block{genesis}, latestState: genesis.state.Copy()}
}

// LatestWorldState returns a copy of the world state stored in the last block
// TODO: do we need a lock?
func (bc *BlockChain) LatestWorldState() storage.KV {
	return bc.latestState
}

// might panic, we need to ensure genesis block always exists
func (bc *BlockChain) lastBlock() *Block {
	return bc.blocks[len(bc.blocks)-1]
}

func (bc *BlockChain) Append(block *Block) error {
	bc.blocks = append(bc.blocks, block)
	bc.latestState = block.state.Copy()
	return nil
}

func (bc *BlockChain) String() string {
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
