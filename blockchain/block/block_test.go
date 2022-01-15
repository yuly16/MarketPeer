package block

import (
	"fmt"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/storage"
	"testing"
)

func TestBlockBuilder(t *testing.T) {
	var kvFactory storage.KVFactory = storage.CreateSimpleKV
	bb := NewBlockBuilder(kvFactory).
		SetParentHash("ffff").
		SetNonce("fuck").
		SetNumber(0).
		//setState(storage.NewSimpleKV()).
		//setTxns(storage.NewSimpleKV()).
		//setReceipts(storage.NewSimpleKV()).
		SetBeneficiary(*account.NewAddress([8]byte{}))
	b := bb.Build()
	fmt.Println(b)

}

func TestBlockBuilder2(t *testing.T) {
	var kvFactory storage.KVFactory = storage.CreateSimpleKV

	// build the genesis block
	addr0 := account.NewAddress([8]byte{0})
	addr1 := account.NewAddress([8]byte{1})
	addr2 := account.NewAddress([8]byte{2})
	addr3 := account.NewAddress([8]byte{3})
	state0 := account.NewStateBuilder(kvFactory).SetBalance(0).Build()
	state1 := account.NewStateBuilder(kvFactory).SetBalance(100).Build()
	state2 := account.NewStateBuilder(kvFactory).SetBalance(200).Build()
	state3 := account.NewStateBuilder(kvFactory).SetBalance(300).Build()

	b1 := NewBlockBuilder(kvFactory).
		SetParentHash("ffff").
		SetNonce("fuck").
		SetNumber(0).
		SetAddrState(addr0, state0).
		SetAddrState(addr1, state1).
		SetAddrState(addr2, state2).
		SetAddrState(addr3, state3).
		SetBeneficiary(*account.NewAddress([8]byte{})).Build()
	fmt.Println(b1)
}

func TestBlockChainString(t *testing.T) {
	var kvFactory storage.KVFactory = storage.CreateSimpleKV

	bb := NewBlockBuilder(kvFactory).
		SetParentHash("ffff").
		SetNonce("fuck").
		SetNumber(0).
		setState(storage.NewSimpleKV()).
		setTxns(storage.NewSimpleKV()).
		setReceipts(storage.NewSimpleKV()).
		SetBeneficiary(*account.NewAddress([8]byte{}))
	b := bb.Build()

	bc := NewBlockChain()
	bc.Append(b)
	bc.Append(b)
	bc.Append(b)
	fmt.Println(bc)
}


func TestBlockChainVerify(t *testing.T) {
	var kvFactory storage.KVFactory = storage.CreateSimpleKV

	bb := NewBlockBuilder(kvFactory).
		SetParentHash("ffff").
		SetNonce("fuck").
		SetNumber(0).
		setState(storage.NewSimpleKV()).
		setTxns(storage.NewSimpleKV()).
		setReceipts(storage.NewSimpleKV()).
		SetBeneficiary(*account.NewAddress([8]byte{}))
	b := bb.Build()

	bc := NewBlockChain()
	bc.Append(b)
	bc.Append(b)
	bc.Append(b)
	fmt.Println(bc)
}

func TestBlockHash(t *testing.T) {
	genesis := DefaultGenesis()
	fmt.Println(genesis.Hash())
	next := NewBlockBuilder(storage.CreateSimpleKV).SetParentHash(genesis.Hash()).Build()
	fmt.Println(next.Hash())
}

