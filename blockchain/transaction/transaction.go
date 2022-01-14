package transaction

import (
	"fmt"
	"go.dedis.ch/cs438/blockchain/account"
)

func NewTransaction() Transaction {
	txn := Transaction{}
	txn.Nonce = 1
	txn.Value = 32
	txn.V = "abc"
	txn.S = "def"
	txn.R = "ghi"
	return txn
}

type Transaction struct {
	Nonce int
	From  account.Address
	To    account.Address
	Value int
	V     string
	R     string
	S     string
}

func (t Transaction) Print() {
	fmt.Printf("transaction info: nonce: %d, value: %d, v: %s, r: %s, s:%s\n",
		t.Nonce, t.Value, t.V, t.R, t.S)
}
