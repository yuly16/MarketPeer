package storage

// BlockChain is a chain of Blocks
type BlockChain struct {
	blocks []*Block // lets first store it in a array
}

func NewBlockChain() *BlockChain {
	return &BlockChain{blocks: make([]*Block, 0)}
}

func (bc *BlockChain) Append(block *Block) error {
	bc.blocks = append(bc.blocks, block)
	return nil
}

func (bc *BlockChain) String() string {

	return ""
}
