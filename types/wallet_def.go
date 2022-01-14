package types

import "go.dedis.ch/cs438/blockchain/transaction"

// WalletTransactionMessage is the transaction message
type WalletTransactionMessage struct {
	// transaction info
	Txn transaction.SignedTransaction
}