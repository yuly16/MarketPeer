package miner

import (
	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/blockchain/transaction"
	"time"
)

// daemons
// TODO: finish verification logic
func (m *Miner) verifyTxnd() {
	// when verified transactions > threshold, we assemble a new block
	// TODO: do we need to maintain current block's state? such that each transaction could modify the state
	//		then, when we assemble the block, we directly set the new state?

	verifiedTxns := make([]*transaction.SignedTransaction, 0, 10)
	for !m.isKilled() {
		txn := <-m.txnCh
		verifiedTxns = append(verifiedTxns, txn)
		if len(verifiedTxns) < m.blocktxns {
			continue
		}
		// could assemble a new block

	}
}

// TODO: onAssembleBlock needs to do PoW
func (m *Miner) onAssembleBlock(verifiedTxns []*transaction.Transaction) *block.Block {
	return block.NewBlockBuilder(m.kvFactory).Build()
}

func (m *Miner) verifyBlockd() {
	for !m.isKilled() {
		time.Sleep(1000)
	}
}
