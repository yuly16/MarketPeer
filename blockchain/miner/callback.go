package miner

import (
	"strings"

	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

// TODO: finish callback
func (m *Miner) WalletTxnMsgCallback(msg types.Message, pkt transport.Packet) error {
	logger := m.logger.With().Str("callback", "TxnMsg").Logger()
	txn := msg.(*types.WalletTransactionMessage)
	logger.Info().Msgf("receive msg: %s", txn.String())
	m.txnCh <- &txn.Txn
	logger.Info().Msgf("txn sent: %s", txn.Txn.String())
	return nil
}

// TODO: race condition between blockd and txnd
// txnd might work on stale last block
// blockd might append to a wrong position
func (m *Miner) BlockMsgCallback(msg types.Message, pkt transport.Packet) error {
	//logger := m.logger.With().Str("callback", "BlockMsg").Logger()
	block := msg.(*types.BlockMessage)
	validate := true
	// number
	// validate previous block
	parentHash := block.Block.Header.ParentHash
	validate = validate && (m.chain.LastBlock().Hash() == parentHash)
	// validate timestamp
	parentTs := m.chain.LastBlock().Header.Timestamp
	validate = validate && (parentTs == block.Block.Header.Timestamp)
	// validate POW
	difficulty := block.Block.Header.Difficulty
	blockHash := block.Block.Hash()
	expect := strings.Repeat("0", difficulty)
	actual := blockHash[:difficulty]
	validate = validate && (strings.Compare(expect, actual) == 0)
	// validate STATE
	// TODO: need smart contract

	// if validate is true, save it
	if validate {
		err := m.chain.Append(&block.Block)
		if err != nil {
			return err
		}
	}

	return nil
}
