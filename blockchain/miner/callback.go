package miner

import (
	"fmt"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"strings"
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

func (m *Miner) BlockMsgCallback(msg types.Message, pkt transport.Packet) error {
	//logger := m.logger.With().Str("callback", "BlockMsg").Logger()
	block := msg.(*types.BlockMessage)
	difficulty := block.Block.Header.Difficulty
	blockHash := block.Block.Hash()
	expect := strings.Repeat("0", difficulty)
	actual := blockHash[:difficulty]
	fmt.Println(expect)
	if strings.Compare(expect, actual) == 0 {
		fmt.Println("jap")
	} else {
		fmt.Println("juai")
	}
	return nil
}