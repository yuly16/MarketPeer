package tests

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/blockchain/storage"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/registry/standard"
	"go.dedis.ch/cs438/transport/channel"
)

func TestBlockBuilder(t *testing.T) {
	var kvFactory storage.KVFactory = storage.CreateSimpleKV
	bb := block.NewBlockBuilder(kvFactory).
		SetParentHash("ffff").
		SetNonce(0).
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

	b1 := block.NewBlockBuilder(kvFactory).
		SetParentHash("ffff").
		SetNonce(0).
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

	bb := block.NewBlockBuilder(kvFactory).
		SetParentHash("ffff").
		SetNonce(0).
		SetNumber(0).
		SetState(storage.NewSimpleKV()).
		SetTxns(storage.NewSimpleKV()).
		SetReceipts(storage.NewSimpleKV()).
		SetBeneficiary(*account.NewAddress([8]byte{}))
	b := bb.Build()

	bc := block.NewBlockChain()
	bc.Append(b)
	bc.Append(b)
	bc.Append(b)
	fmt.Println(bc)
}

func TestBlockChainVerify(t *testing.T) {
	var kvFactory storage.KVFactory = storage.CreateSimpleKV
	transp := channel.NewTransport()
	sock1, err := transp.CreateSocket("127.0.0.1:0")

	require.NoError(t, err)
	privateKey1, err := crypto.GenerateKey()
	require.NoError(t, err)

	fullNode1, _ := z.NewTestFullNode(t,
		z.WithSocket(sock1),
		z.WithMessageRegistry(standard.NewRegistry()),
		z.WithPrivateKey(privateKey1),
	)
	fullNode1.Start()
	defer fullNode1.Stop()

	bb := block.NewBlockBuilder(kvFactory).
		SetParentHash("ffff").
		SetNonce(1).
		SetNumber(0).
		SetState(storage.NewSimpleKV()).
		SetTxns(storage.NewSimpleKV()).
		SetDifficulty(2).
		SetReceipts(storage.NewSimpleKV()).
		SetBeneficiary(*account.NewAddress([8]byte{}))
	b := bb.Build()
	//fullNode1.Test_submitTxn()
	bc := block.NewBlockChain()
	bc.Append(b)
	bc.Append(b)
	bc.Append(b)
	fmt.Println(bc)
	fullNode1.BroadcastBlock(*b)
	time.Sleep(time.Second * 5)
}

func TestBlockHash(t *testing.T) {
	genesis := block.DefaultGenesis()
	fmt.Println(genesis.Hash())
	next := block.NewBlockBuilder(storage.CreateSimpleKV).SetParentHash(genesis.Hash()).Build()
	fmt.Println(next.Hash())
}

func TestBlockMarshal(t *testing.T) {
	genesis := DefaultGenesis()
	fmt.Println(genesis.String(), genesis.Hash())

	v, err := json.Marshal(genesis)
	fmt.Println(string(v))
	require.NoError(t, err)
	//unmarshaled := &Block{}
	unmarshaled := NewBlockBuilder(storage.CreateSimpleKV).Build()
	json.Unmarshal(v, unmarshaled)
	//fmt.Println(unmarshaled.String(), unmarshaled.Hash())

	fmt.Println(unmarshaled.String())
}
