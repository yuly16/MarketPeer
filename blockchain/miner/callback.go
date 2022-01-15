package miner

import (
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
