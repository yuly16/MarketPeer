package miner

import (
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

func (m *Miner) WalletTxnMsgCallback(msg types.Message, pkt transport.Packet) error {
	logger := m.logger.With().Str("callback", "TxnMsg").Logger()
	txn := msg.(*types.WalletTransactionMessage)
	logger.Info().Msgf("receive txn msg: %s", txn.String())
	m.txnCh <- &txn.Txn
	logger.Debug().Msgf("txn sent to txnVerifyd: %s", txn.Txn.String())
	return nil
}

func (m *Miner) BlockMsgCallback(msg types.Message, pkt transport.Packet) error {
	//logger := m.logger.With().Str("callback", "BlockMsg").Logger()
	logger := m.logger.With().Str("callback", "BlockMsg").Logger()

	blockMsg := msg.(*types.BlockMessage)
	raw := blockMsg.Block
	reconstruct := block.NewBlockBuilder(m.kvFactory)
	reconstruct.SetTxns(raw.Transactions).SetHeader(raw.Header)
	accounts := m.kvFactory()
	// rebuild account state; TODO, make it a function of block
	err := raw.State.For(func(key string, value interface{}) error {
		// key is account.Address, value is serialized form of account.State

		vv, ok := value.(map[string]interface{}) // vv is serialized form of account.State
		if !ok {
			panic(value)
		}
		accountState := account.NewStateBuilder(m.kvFactory)
		accountState.SetNonce(uint(vv["Nonce"].(float64))).
			SetBalance(uint(vv["Balance"].(float64))).
			SetCode(vv["Code"].(string))

		vvs, ok := vv["StorageRoot"].(map[string]interface{}) // vvs is serialized account.State.StorageRoot
		if !ok {
			panic(vvs)
		}
		vvsi, ok := vvs["Internal"].(map[string]interface{}) // vvsi storageRoot.internal
		if !ok {
			panic(vvsi)
		}
		for vvsik, vvsiv := range vvsi {
			accountState.SetKV(vvsik, vvsiv)
		}

		if err := accounts.Put(key, accountState.Build()); err != nil {
			panic(err)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	reconstruct.SetWorldState(accounts)

	blockMsg.Block = *reconstruct.Build()
	logger.Info().Msgf("receive block msg: %s", blockMsg.String())

	m.blockCh <- &blockMsg.Block

	return nil
}
