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

type VerifyTransactionMessage struct {
	// transaction handle
	NetworkAddr string
	Handle      transaction.SignedTransactionHandle
}

type VerifyTransactionReplyMessage struct {
	// transaction handle
	Handle      transaction.SignedTransactionHandle
	BlocksAfter int // #blocks after the block including the handle. -1 means txn not in chain
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
