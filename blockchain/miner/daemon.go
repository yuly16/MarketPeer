package miner

import (
	"fmt"
	"time"

	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/blockchain/transaction"
	"go.dedis.ch/cs438/types"
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
		m.logger.Debug().Msgf("block constructed, prepare to do PoW: %s", bb.Build().String())
		finalized := m.blockPoW(bb)

		// TODO: set the receipts
		return finalized
	}
}

// Q: what if there are some old transactions?
// A: txn.nonce is compared with account.nonce

// daemons
func (m *Miner) verifyAndExecuteTxnd() {
	// when verified transactions > threshold, we assemble a new block
	for !m.isKilled() {
		// Q: what if the latest block is changed?
		// A: then the append will fail, it will not affect the consistency
		// return a future, which tells us if the latestState has changed
		latestState, lastBlock, err := m.chain.LatestWorldState() // based on the latest state in our chain
		for err != nil {
			// TODO: it might be better to trigger-based
			time.Sleep(100)
			latestState, lastBlock, err = m.chain.LatestWorldState() // based on the latest state in our chain

		}
		// now we get the latest state
		newBlock := m.verifyAndExecuteTxnLoop(lastBlock, latestState)
		m.logger.Debug().Msgf("verify and execute loop done once")

		// Note: append will be done by callback? TODO
		//// append the newBlock to the blockchain
		//if err := m.chain.Append(newBlock); err != nil {
		//	m.logger.Err(fmt.Errorf("mining error: %w", err)).Send()
		//	continue
		//}
		//m.logger.Info().Msgf("append the new Block to chain")

		// broadcast the newBlock to others
		if _, err := m.chain.TryAppend(newBlock); err != nil {
			m.logger.Warn().Msgf(fmt.Sprintf("mined block is stale, cannot append: %s", newBlock.String()))
			// now check if the transactions in block are executed, if not, then rebuild block
			// although this is an invalid block, but the txns in yet might not be included in current chain
			// if this is not added, 3 nodes submit 3 concurrent txns. some txns might not be included in the network
			for _, txn := range newBlock.Transactions {
				m.txnCh <- txn
			}
			continue
		}
		m.logger.Info().Msgf("PoW done, block: %s", newBlock.String())
		msg := types.BlockMessage{*newBlock}
		if err := m.messaging.Broadcast(msg); err != nil {
			m.logger.Err(err).Send()
			continue
		}
		m.logger.Info().Msgf("block broadcasted")

	}
}

// TODO: race condition between blockd and txnd
// txnd might work on stale last block
// blockd might append to a wrong position
// Q: what if number is larger? A: handled by TryAppend and Append
// Q: what if a stale block? A: handled by TryAppend and Append
func (m *Miner) verifyBlockd() {
	for !m.isKilled() {
		newBlock := <-m.blockCh
		m.logger.Debug().Msgf("begin verify the block: %s", newBlock.String())
		// first check if it is a valid PoW block
		// validate POW
		difficulty := newBlock.Header.Difficulty
		powValid := satisfyPrefixZeros(newBlock.HashBytes(), difficultyToZeros(difficulty))
		if !powValid {
			m.logger.Warn().Msgf("block pow not valid, block=%s", newBlock.String())
			continue
		}
		// then try to append, the append will tell us if it can append
		parent, err := m.chain.TryAppend(newBlock)
		if err != nil {
			m.logger.Err(err).Send()
			continue
		}
		// now it could append, we check if after execute, the state is consistent
		parentState := parent.State.Copy()
		for _, txn := range newBlock.Transactions {
			if err := m.executeTxn(txn, parentState); err != nil {
				m.logger.Err(fmt.Errorf("replay txns error: %w", err)).Send()
				continue
			}
		}
		// check if after replay the state is the same
		if parentState.Hash() != newBlock.Header.StateHash {
			m.logger.Err(fmt.Errorf("after replay state not same: expect=%s, given=%s",
				parentState, newBlock.State)).Send()
			continue
		}
		// now we could finally append
		if err := m.chain.Append(newBlock); err != nil {
			m.logger.Err(fmt.Errorf("though all previous validation passed, " +
				"but the chain has changed, so we cannot append")).Send()
			continue
		}
		m.logger.Info().Msgf("verify done, block valid and appended: %s", newBlock.String())

	}
}
