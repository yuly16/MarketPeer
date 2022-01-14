package block

import (
	"fmt"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/storage"
	"testing"
)

func TestBlockBuilder(t *testing.T) {
	bb := NewBlockBuilder().
		setParentHash("ffff").
		setNonce("fuck").
		setNumber(0).
		setState(storage.NewSimpleKV()).
		setTxns(storage.NewSimpleKV()).
		setReceipts(storage.NewSimpleKV()).
		setBeneficiary(*account.NewAddress([20]byte{}))
	b := bb.build()
	fmt.Println(b)

}

func TestBlockChainString(t *testing.T) {
	bb := NewBlockBuilder().
		setParentHash("ffff").
		setNonce("fuck").
		setNumber(0).
		setState(storage.NewSimpleKV()).
		setTxns(storage.NewSimpleKV()).
		setReceipts(storage.NewSimpleKV()).
		setBeneficiary(*account.NewAddress([20]byte{}))
	b := bb.build()

	bc := NewBlockChain()
	bc.Append(b)
	bc.Append(b)
	bc.Append(b)
	fmt.Println(bc)
}
