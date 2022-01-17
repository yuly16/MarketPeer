package miner

import (
	"fmt"
	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/types"
	"time"
)

func (m *Miner) askForMissingBlocks(to string, lastNumber int) ([]*block.Block, error) {
	askNumber := lastNumber
	blocks := make([]*block.Block, 0, lastNumber+1)
	for askNumber >= 0 {
		b, err := m.askForBlock(to, askNumber)
		if err != nil {
			return nil, err
		}
		// test if b can be connect to our network
		// Now we simply do a bruteforce, which will ask for all the blocks. FIXME
		blocks = append(blocks, b)
		askNumber -= 1
	}

	return blocks, nil
}

func (m *Miner) askForBlock(to string, number int) (*block.Block, error) {
	stamp := int(time.Now().UnixMilli())
	future := make(chan *types.AskForBlockReplyMessage, 1)
	m.askForBlocksFutures[stamp] = future
	ask := &types.AskForBlockMessage{m.addr, stamp, number}
	if err := m.messaging.Unicast(to, ask); err != nil {
		return nil, fmt.Errorf("ask for block error: %w", err)
	}
	reply := <-future
	if !reply.Success {
		return nil, fmt.Errorf("ask for block error: target=%s dont have this block(%d)", to, number)
	}
	return &reply.Block, nil
}
