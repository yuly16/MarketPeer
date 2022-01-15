package miner

import (
	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/blockchain/transaction"
	"time"
)

// verifyTxnLoop will verify  a block of transactions,
// until it reached the blockTxns threshold. based on the given worldState
func (m *Miner) verifyTxnLoop(worldState storage.KV) *block.Block {
	verifiedTxns := make([]*transaction.SignedTransaction, 0, 10)

	for {
		txn := <-m.txnCh
		if err := m.verifyTxn(txn, worldState); err != nil {
			m.logger.Warn().Msgf("verify failed, err=%v", err)
			continue
		}
		verifiedTxns = append(verifiedTxns, txn)
		if len(verifiedTxns) < m.blocktxns {
			continue
		}
		m.logger.Info().Msgf("%d verfied txns, prepare to construct the block", len(verifiedTxns))
		block := block.NewBlockBuilder(m.kvFactory)
		return block.Build()
	}
	return nil
}

// daemons
// TODO: finish verification logic
func (m *Miner) verifyTxnd() {
	// when verified transactions > threshold, we assemble a new block
	// TODO: do we need to maintain current block's state? such that each transaction could modify the state
	//		then, when we assemble the block, we directly set the new state?
	//latestState := m.chain.LatestWorldState() // we are mutating the latest state in our chain
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
