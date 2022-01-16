package wallet

import (
	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

func (w *Wallet) SyncAccountReplyMessageCallback(msg types.Message, pkt transport.Packet) error {
	logger := w.logger.With().Str("callback", "SyncAccountReply").Logger().Level(zerolog.ErrorLevel)
	syncMsg := msg.(*types.SyncAccountReplyMessage)
	logger.Info().Msgf("receive sync account reply msg: %s", syncMsg.String())

	future, ok := w.syncFutures[syncMsg.Timestamp]
	if !ok {
		logger.Warn().Msgf("msg has no future: %s", syncMsg.String())
		return nil
	}
	select {
	case future <- syncMsg:
	default:
	}
	return nil
}

func (w *Wallet) VerifyTxnReplyMessageCallback(msg types.Message, pkt transport.Packet) error {
	logger := w.logger.With().Str("callback", "VerifyTransactionReply").Logger().Level(zerolog.ErrorLevel)
	verifyMsg := msg.(*types.VerifyTransactionReplyMessage)
	logger.Info().Msgf("receive verify txn reply msg: %s", verifyMsg.String())

	future, ok := w.verifyFutures[verifyMsg.Handle]
	if !ok {
		logger.Warn().Msgf("msg has no future: %s", verifyMsg.String())
		return nil
	}
	select {
	case future <- verifyMsg:
	default:
	}
	return nil
}
