package miner

import (
	"fmt"
	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/blockchain/transaction"
	"time"
)

// verifyTxnLoop will verify  a block of transactions,
// until it reached the blockTxns threshold. based on the given worldState
func (m *Miner) verifyAndExecuteTxnLoop(lastBlock *block.Block, worldState storage.KV) *block.Block {
	verifiedTxns := make([]*transaction.SignedTransaction, 0, m.blocktxns+1)
	for {
		
		txn := <-m.txnCh
		// verify
		if err := m.verifyTxn(txn, worldState); err != nil {
			m.logger.Warn().Msgf("verify failed, err=%v", err)
			continue
		}
		// execute
		if err := m.executeTxn(txn, worldState); err != nil {
			m.logger.Warn().Msgf("execute failed, err=%v", err)
			continue
		}
		m.logger.Info().Msgf("verify and execute txn: %s", txn.String())

		verifiedTxns = append(verifiedTxns, txn)
		if len(verifiedTxns) < m.blocktxns {
			continue
		}
		m.logger.Info().Msgf("%d verfied txns, prepare to construct the block", len(verifiedTxns))

		// create a new block and broadcast to other miners
		// first create basic block template based on last block
		// state, transactions, and receipts remain to set
		// nonce remain to compute
		bb := lastBlock.NextBlockBuilder(m.kvFactory, m.accountAddr)
		// set the state
		bb.SetWorldState(worldState)
		// set the txns
		for _, txn := range verifiedTxns {
			bb.AddTxn(txn)
		}
		// TODO: set the receipts
		return m.blockPoW(bb)
	}
}

// daemons
// TODO: finish verification logic
func (m *Miner) verifyAndExecuteTxnd() {
	// when verified transactions > threshold, we assemble a new block
	for !m.isKilled() {
		// TODO: what if the latest block is changed?
		latestState, lastBlock := m.chain.LatestWorldState() // based on the latest state in our chain
		newBlock := m.verifyAndExecuteTxnLoop(lastBlock, latestState)
		// broadcast the newBlock to others
		fmt.Println(newBlock)
	}
}

func (m *Miner) verifyBlockd() {
	for !m.isKilled() {
		time.Sleep(1000)
	}
}
