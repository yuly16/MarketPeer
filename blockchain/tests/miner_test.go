package tests

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
	"go.dedis.ch/cs438/blockchain"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/blockchain/storage"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/registry/standard"
	"go.dedis.ch/cs438/transport/channel"

	z "go.dedis.ch/cs438/internal/testing"
)

// based on init accounts info, generate the genesis block
func generateGenesisBlock(kvFactory storage.KVFactory, accounts ...*account.Account) *block.Block {
	bb := block.NewBlockBuilder(kvFactory).
		SetParentHash(block.DUMMY_PARENT_HASH).
		SetNonce("fuck").
		SetNumber(0).
		SetBeneficiary(*account.NewAddress([8]byte{}))
	for _, acc := range accounts {
		bb.SetAddrState(acc.GetAddr(), acc.GetState())
	}
	b := bb.Build()
	return b
}

func TestBuildFullNode(t *testing.T) {
	// init some network nodes, which have some balance
	transp := channel.NewTransport()
	sock, err := transp.CreateSocket("127.0.0.1:0")
	require.NoError(t, err)
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	publicKey := &privateKey.PublicKey
	ac := account.NewAccountBuilder(crypto.FromECDSAPub(publicKey), storage.CreateSimpleKV).
		WithBalance(100).WithKV("apple", 1).Build()
	genesis := generateGenesisBlock(storage.CreateSimpleKV, ac)

	fullNode, _ := z.NewTestFullNode(t,
		z.WithSocket(sock),
		z.WithMessageRegistry(standard.NewRegistry()),
		z.WithPrivateKey(privateKey),
		z.WithKVFactory(storage.CreateSimpleKV),
		z.WithAccount(ac),
		z.WithGenesisBlock(genesis),
	)
	fullNode.Start()

	time.Sleep(1 * time.Second)
	fullNode.Stop()

}

// 3 full nodes, related to 3 accounts. blockchain only one block
func TestNetworkInit(t *testing.T) {
	accountFactory := func(balance uint, key string, value interface{}) (*account.Account, *ecdsa.PrivateKey) {
		privateKey1, err := crypto.GenerateKey()
		require.NoError(t, err)
		publicKey1 := &privateKey1.PublicKey
		ac1 := account.NewAccountBuilder(crypto.FromECDSAPub(publicKey1), storage.CreateSimpleKV).
			WithBalance(balance).WithKV(key, value).Build()
		return ac1, privateKey1
	}

	acc1, pri1 := accountFactory(100, "apple", 10)
	acc2, pri2 := accountFactory(200, "orange", 20)
	acc3, pri3 := accountFactory(300, "cola", 100)

	genesisFactory := func() *block.Block {
		return generateGenesisBlock(storage.CreateSimpleKV, acc1, acc2, acc3)
	}

	nodeFactory := func(acc *account.Account, pri *ecdsa.PrivateKey) *blockchain.FullNode {
		transp := channel.NewTransport()
		sock, err := transp.CreateSocket("127.0.0.1:0")
		require.NoError(t, err)
		//crypto.
		fullNode, _ := z.NewTestFullNode(t,
			z.WithSocket(sock),
			z.WithMessageRegistry(standard.NewRegistry()),
			z.WithPrivateKey(pri),
			z.WithAccount(acc),
			z.WithGenesisBlock(genesisFactory()),
		)
		return fullNode
	}

	node1 := nodeFactory(acc1, pri1)
	node2 := nodeFactory(acc2, pri2)
	node3 := nodeFactory(acc3, pri3)

	node1.Start()
	node2.Start()
	node3.Start()

	time.Sleep(1 * time.Second)
	node1.Stop()
	node2.Stop()
	node3.Stop()

}

func TestOneConsensus(t *testing.T) {
	// a wallet
}
