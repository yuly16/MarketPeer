package block

// BlockChain is a chain of Blocks
type BlockChain struct {
	blocks []*Block // let's first store it in an array
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
