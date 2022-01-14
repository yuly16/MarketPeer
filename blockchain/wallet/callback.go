package wallet

import (
	"fmt"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

func (w Wallet) WalletTxnMsgCallback(msg types.Message, pkt transport.Packet) error {
	txn := msg.(*types.WalletTransactionMessage)
	fmt.Println(w.addr)
	txn.Txn.Print()
	return nil
}