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
