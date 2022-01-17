package types

import "go.dedis.ch/cs438/blockchain/block"

type BlockMessage struct {
	From  string
	Block block.Block
}

type AskForBlockMessage struct {
	From      string
	TimeStamp int // async notify
	Number    int // inquired block's number
}

type AskForBlockReplyMessage struct {
	TimeStamp int // async notify
	Block     block.Block
	Success   bool // false means we dont have required block TODO: it is possible to fail, our blocks can also be overwritten
}
