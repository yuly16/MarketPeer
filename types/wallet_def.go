package types

import (
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/transaction"
)

// WalletTransactionMessage is the transaction message
type WalletTransactionMessage struct {
	// transaction info
	Txn transaction.SignedTransaction
}

type SyncAccountMessage struct {
	Timestamp   int // used as async notify ID
	NetworkAddr string
	Addr        account.Address
}

type SyncAccountReplyMessage struct {
	Timestamp int // used as async notify ID
	State     account.State
}
