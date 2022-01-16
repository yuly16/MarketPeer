package tests

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/blockchain"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/blockchain/messaging"
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/blockchain/transaction"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/registry/standard"
	"go.dedis.ch/cs438/transport/channel"
	"testing"
	"time"
)

// 3 full nodes, related to 3 accounts.
// blockchain with genesis state:
// 0: 100, apple->10
// 1: 200, orange->20
// 2: 300, cola->100

func TestSubmitTxn(t *testing.T) {
	transp := channel.NewTransport()

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

	nodeFactory := func(acc *account.Account, pri *ecdsa.PrivateKey) (*blockchain.FullNode, messaging.Messager) {
		sock, err := transp.CreateSocket("127.0.0.1:0")
		require.NoError(t, err)
		//crypto.
		fullNode, messager := z.NewTestFullNode(t,
			z.WithSocket(sock),
			z.WithMessageRegistry(standard.NewRegistry()),
			z.WithPrivateKey(pri),
			z.WithAccount(acc),
			z.WithGenesisBlock(genesisFactory()),
		)
		return fullNode, messager
	}

	node1, _ := nodeFactory(acc1, pri1)
	node2, _ := nodeFactory(acc2, pri2)
	node3, _ := nodeFactory(acc3, pri3)
	node1.AddPeer(node2, node3)
	node2.AddPeer(node1, node3)
	node3.AddPeer(node1, node2)

	node1.Start()
	node2.Start()
	node3.Start()
	time.Sleep(200 * time.Millisecond)

	txn := transaction.NewTransaction(0, 5, *node1.GetAccountAddr(), *node2.GetAccountAddr())
	node1.SubmitTxn(txn)

	time.Sleep(10 * time.Second)
	fmt.Println(node1.GetChain())
	fmt.Println(node2.GetChain())

	fmt.Println(node3.GetChain())

	node1.Stop()
	node2.Stop()
	node3.Stop()

}
