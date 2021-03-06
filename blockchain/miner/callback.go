package miner

import (
	"fmt"
	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

func (m *Miner) WalletTxnMsgCallback(msg types.Message, pkt transport.Packet) error {

	logger := m.logger.With().Str("callback", "TxnMsg").Logger()
	txn := msg.(*types.WalletTransactionMessage)
	logger.Info().Msgf("receive txn msg: %s", txn.String())
	if m.attacker && pkt.Header.Source != m.addr {
		logger.Info().Msgf("attacker dont process txns from other addrs")
		return nil
	}
	m.txnCh <- &txn.Txn
	logger.Debug().Msgf("txn sent to txnVerifyd: %s", txn.Txn.String())
	return nil
}

func (m *Miner) BlockMsgCallback(msg types.Message, pkt transport.Packet) error {
	//logger := m.logger.With().Str("callback", "BlockMsg").Logger()
	logger := m.logger.With().Str("callback", "BlockMsg").Logger()
	if m.attacker && pkt.Header.Source != m.addr {
		logger.Info().Msgf("attacker dont process block from other miners")
		return nil
	}
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

	m.blockCh <- blockMsg

	return nil
}

func (m *Miner) SyncMsgCallback(msg types.Message, pkt transport.Packet) error {
	logger := m.logger.With().Str("callback", "SyncAccount").Logger().Level(zerolog.ErrorLevel)

	syncMsg := msg.(*types.SyncAccountMessage)
	addr := syncMsg.Addr
	logger.Info().Msgf("receive sync account msg: %s", syncMsg.String())
	// fetch latest state of this addr
	worldState, _, err := m.chain.LatestWorldState()
	if err != nil {
		err = fmt.Errorf("sync msg error: %w", err)
		logger.Err(err).Send()
		return err
	}
	v, err := worldState.Get(addr.String())
	if err != nil {
		err = fmt.Errorf("sync msg error: %w", err)
		logger.Err(err).Send()
		return err
	}
	state, ok := v.(*account.State)
	if !ok {
		err = fmt.Errorf("account state=%s cannot casted to *State", v)
		logger.Err(err).Send()
		return err
	}

	reply := &types.SyncAccountReplyMessage{Timestamp: syncMsg.Timestamp, State: *state}
	if err = m.messaging.Unicast(syncMsg.NetworkAddr, reply); err != nil {
		err = fmt.Errorf("sync msg error: %w", err)
		logger.Err(err).Send()
		return err
	}
	logger.Info().Msgf("send back sync reply=%s to %s", reply.String(), syncMsg.NetworkAddr)

	return nil
}

func (m *Miner) VerifyTxnMsgCallback(msg types.Message, pkt transport.Packet) error {
	logger := m.logger.With().Str("callback", "VerifyTransaction").Logger().Level(zerolog.ErrorLevel)

	verifyMsg := msg.(*types.VerifyTransactionMessage)
	handle := verifyMsg.Handle
	_, nBlocksAfter := m.chain.HasTxn(handle)
	reply := &types.VerifyTransactionReplyMessage{handle, nBlocksAfter}
	if err := m.messaging.Unicast(verifyMsg.NetworkAddr, reply); err != nil {
		err = fmt.Errorf("verify msg callback error: %w", err)
		logger.Err(err).Send()
		return err
	}
	logger.Info().Msgf("send back verify txn reply=%s to %s", reply.String(), verifyMsg.NetworkAddr)
	return nil
}

func (m *Miner) AskForBlockMsgCallback(msg types.Message, pkt transport.Packet) error {
	logger := m.logger.With().Str("callback", "AskForBlock").Logger().Level(zerolog.InfoLevel)
	ask := msg.(*types.AskForBlockMessage)
	logger.Info().Msgf("receive msg: %s", ask.String())
	b, err := m.chain.GetBlock(ask.Number)
	var reply *types.AskForBlockReplyMessage
	if err != nil {
		reply = &types.AskForBlockReplyMessage{
			TimeStamp: ask.TimeStamp,
			Block:     block.Block{},
			Success:   false,
		}
		logger.Warn().Msgf("cannot get block(%d), reason=%v", ask.Number, err)
	} else {
		reply = &types.AskForBlockReplyMessage{
			TimeStamp: ask.TimeStamp,
			Block:     *b,
			Success:   true,
		}
		logger.Info().Msgf("get block(%d)", ask.Number)
	}
	if err := m.messaging.Unicast(ask.From, reply); err != nil {
		return fmt.Errorf("ask for block msg callback error: %w", err)
	}
	return nil
}

func (m *Miner) AskForBlockReplyMsgCallback(msg types.Message, pkt transport.Packet) error {
	logger := m.logger.With().Str("callback", "VerifyTransaction").Logger().Level(zerolog.InfoLevel)
	reply := msg.(*types.AskForBlockReplyMessage)
	logger.Info().Msgf("receive msg: %s", reply.String())

	future, ok := m.askForBlocksFutures[reply.TimeStamp]
	if !ok {
		logger.Warn().Msgf("msg has no future: %s", reply.String())
		return nil
	}
	select {
	case future <- reply:
	default:
	}
	return nil
}
