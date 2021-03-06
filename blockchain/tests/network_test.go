package tests

import (
	"crypto/ecdsa"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/blockchain"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/blockchain/transaction"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/registry/standard"
	"go.dedis.ch/cs438/transport/channel"
)

// 3 full nodes, related to 3 accounts.
// blockchain with genesis state:
// 0: 100, apple->10
// 1: 200, orange->20
// 2: 300, cola->100
func Test_Network_SubmitTxn1(t *testing.T) {
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

	nodeFactory := func(acc *account.Account, pri *ecdsa.PrivateKey) *blockchain.FullNode {
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
	fmt.Printf("node1 chain: \n%s", node1.GetChain())
	fmt.Printf("node2 chain: \n%s", node2.GetChain())
	fmt.Printf("node3 chain: \n%s", node3.GetChain())

	node1.Stop()
	node2.Stop()
	node3.Stop()

}

// each node send a transaction
func Test_Network_SubmitTxn2(t *testing.T) {
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

	nodeFactory := func(acc *account.Account, pri *ecdsa.PrivateKey) *blockchain.FullNode {
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
	node1.AddPeer(node2, node3)
	node2.AddPeer(node1, node3)
	node3.AddPeer(node1, node2)

	node1.Start()
	node2.Start()
	node3.Start()
	time.Sleep(200 * time.Millisecond)

	txn1 := transaction.NewTransaction(0, 5, *node1.GetAccountAddr(), *node2.GetAccountAddr())
	node1.SubmitTxn(txn1)
	txn2 := transaction.NewTransaction(0, 5, *node2.GetAccountAddr(), *node3.GetAccountAddr())
	node2.SubmitTxn(txn2)
	txn3 := transaction.NewTransaction(0, 5, *node3.GetAccountAddr(), *node1.GetAccountAddr())
	node3.SubmitTxn(txn3)

	time.Sleep(3 * time.Second)
	fmt.Printf("node1 chain: \n%s", node1.GetChain())
	fmt.Printf("node2 chain: \n%s", node2.GetChain())
	fmt.Printf("node3 chain: \n%s", node3.GetChain())

	require.Equal(t, node1.GetChain().Hash(), node2.GetChain().Hash())
	require.Equal(t, node2.GetChain().Hash(), node3.GetChain().Hash())
	go func() {
		node1.SyncAccount()
	}()
	go func() { node2.SyncAccount() }()
	go func() { node3.SyncAccount() }()
	time.Sleep(3 * time.Second)
	require.Equal(t, acc1.String(), node1.GetAccount().String())
	require.Equal(t, acc2.String(), node2.GetAccount().String())
	require.Equal(t, acc3.String(), node3.GetAccount().String())

	node1.Stop()
	node2.Stop()
	node3.Stop()
}

// 5 nodes
func Test_Network_SubmitTxn3(t *testing.T) {
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
	acc4, pri4 := accountFactory(400, "laptop", 100)
	acc5, pri5 := accountFactory(500, "iphone", 100)

	genesisFactory := func() *block.Block {
		return generateGenesisBlock(storage.CreateSimpleKV, acc1, acc2, acc3, acc4, acc5)
	}

	nodeFactory := func(acc *account.Account, pri *ecdsa.PrivateKey) *blockchain.FullNode {
		sock, err := transp.CreateSocket("127.0.0.1:0")
		require.NoError(t, err)
		//crypto.
		fullNode, _ := z.NewTestFullNode(t,
			z.WithSocket(sock),
			z.WithMessageRegistry(standard.NewRegistry()),
			z.WithPrivateKey(pri),
			z.WithAccount(acc),
			z.WithGenesisBlock(genesisFactory()),
			z.WithAntiEntropy(100*time.Millisecond),
		)
		return fullNode
	}

	node1 := nodeFactory(acc1, pri1)
	node2 := nodeFactory(acc2, pri2)
	node3 := nodeFactory(acc3, pri3)
	node4 := nodeFactory(acc4, pri4)
	node5 := nodeFactory(acc5, pri5)

	node1.AddPeer(node2, node3, node4)
	node2.AddPeer(node1, node3, node5)
	node3.AddPeer(node1, node2)
	node4.AddPeer(node1, node5)
	node5.AddPeer(node4, node2)

	node1.Start()
	node2.Start()
	node3.Start()
	node4.Start()
	node5.Start()
	time.Sleep(200 * time.Millisecond)

	txn1 := transaction.NewTransaction(0, 5, *node1.GetAccountAddr(), *node2.GetAccountAddr())
	node1.SubmitTxn(txn1)
	time.Sleep(200 * time.Millisecond)

	txn2 := transaction.NewTransaction(0, 5, *node2.GetAccountAddr(), *node3.GetAccountAddr())
	node2.SubmitTxn(txn2)
	time.Sleep(200 * time.Millisecond)

	txn3 := transaction.NewTransaction(0, 5, *node3.GetAccountAddr(), *node1.GetAccountAddr())
	node3.SubmitTxn(txn3)
	time.Sleep(200 * time.Millisecond)

	txn4 := transaction.NewTransaction(0, 5, *node4.GetAccountAddr(), *node5.GetAccountAddr())
	node4.SubmitTxn(txn4)
	time.Sleep(200 * time.Millisecond)

	txn5 := transaction.NewTransaction(0, 5, *node5.GetAccountAddr(), *node4.GetAccountAddr())
	node5.SubmitTxn(txn5)
	time.Sleep(200 * time.Millisecond)

	time.Sleep(10 * time.Second)
	fmt.Printf("node2 chain: \n%s", node2.GetChain())
	fmt.Printf("node3 chain: \n%s", node3.GetChain())
	//fmt.Printf("node5 chain: \n%s", node1.GetChain())

	//fmt.Printf("node2 chain: \n%s", node2.GetChain())
	//fmt.Printf("node3 chain: \n%s", node3.GetChain())

	require.Equal(t, node1.GetChain().Hash(), node2.GetChain().Hash())
	require.Equal(t, node2.GetChain().Hash(), node3.GetChain().Hash())
	require.Equal(t, node3.GetChain().Hash(), node4.GetChain().Hash())
	require.Equal(t, node4.GetChain().Hash(), node5.GetChain().Hash())

	require.Equal(t, node1.GetChain().String(), node2.GetChain().String())
	require.Equal(t, node2.GetChain().String(), node3.GetChain().String())
	require.Equal(t, node3.GetChain().String(), node4.GetChain().String())
	require.Equal(t, node4.GetChain().String(), node5.GetChain().String())

	go func() {
		node1.SyncAccount()
	}()
	go func() { node2.SyncAccount() }()
	go func() { node3.SyncAccount() }()

	go func() { node4.SyncAccount() }()
	go func() { node5.SyncAccount() }()
	time.Sleep(3 * time.Second)
	require.Equal(t, acc1.String(), node1.GetAccount().String())
	require.Equal(t, acc2.String(), node2.GetAccount().String())
	require.Equal(t, acc3.String(), node3.GetAccount().String())
	require.Equal(t, acc4.String(), node4.GetAccount().String())
	require.Equal(t, acc5.String(), node5.GetAccount().String())
	node1.Stop()
	node2.Stop()
	node3.Stop()
	node4.Stop()
	node5.Stop()
}

// 3 full nodes, related to 3 accounts.
// blockchain with genesis state:
// 0: 100, apple->10
// 1: 200, orange->20
// 2: 300, cola->100
// node 0, send 3 txns. only first is in block, since the nonce is wrong
func Test_Network_SendTxns_Invalid(t *testing.T) {
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

	nodeFactory := func(acc *account.Account, pri *ecdsa.PrivateKey) *blockchain.FullNode {
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
	node1.AddPeer(node2, node3)
	node2.AddPeer(node1, node3)
	node3.AddPeer(node1, node2)

	node1.Start()
	node2.Start()
	node3.Start()
	time.Sleep(200 * time.Millisecond)

	txn1 := transaction.NewTransaction(0, 5, *node1.GetAccountAddr(), *node2.GetAccountAddr())
	node1.SubmitTxn(txn1)
	time.Sleep(200 * time.Millisecond)

	txn2 := transaction.NewTransaction(0, 10, *node1.GetAccountAddr(), *node2.GetAccountAddr())
	node1.SubmitTxn(txn2)
	time.Sleep(200 * time.Millisecond)

	txn3 := transaction.NewTransaction(0, 20, *node1.GetAccountAddr(), *node2.GetAccountAddr())
	node1.SubmitTxn(txn3)
	time.Sleep(200 * time.Millisecond)

	time.Sleep(5 * time.Second)
	require.Equal(t, node1.GetChain().Hash(), node2.GetChain().Hash())
	require.Equal(t, node2.GetChain().Hash(), node3.GetChain().Hash())
	require.Equal(t, 2, node1.GetChain().Len())

	fmt.Printf("node1 chain: \n%s", node1.GetChain())

}

func Test_Network_SendTxns_Valid(t *testing.T) {
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

	nodeFactory := func(acc *account.Account, pri *ecdsa.PrivateKey) *blockchain.FullNode {
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
	node1.AddPeer(node2, node3)
	node2.AddPeer(node1, node3)
	node3.AddPeer(node1, node2)

	node1.Start()
	node2.Start()
	node3.Start()
	time.Sleep(200 * time.Millisecond)

	txn1 := transaction.NewTransaction(0, 5, *node1.GetAccountAddr(), *node2.GetAccountAddr())
	node1.SubmitTxn(txn1)
	time.Sleep(200 * time.Millisecond)

	txn2 := transaction.NewTransaction(1, 10, *node1.GetAccountAddr(), *node2.GetAccountAddr())
	node1.SubmitTxn(txn2)
	time.Sleep(200 * time.Millisecond)

	txn3 := transaction.NewTransaction(2, 20, *node1.GetAccountAddr(), *node2.GetAccountAddr())
	node1.SubmitTxn(txn3)
	time.Sleep(200 * time.Millisecond)

	time.Sleep(5 * time.Second)
	require.Equal(t, node1.GetChain().Hash(), node2.GetChain().Hash())
	require.Equal(t, node2.GetChain().Hash(), node3.GetChain().Hash())
	require.Equal(t, 4, node1.GetChain().Len())
	fmt.Printf("node1 chain: \n%s", node1.GetChain())
}

func Test_Network_Transfer_Epfer(t *testing.T) {
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

	nodeFactory := func(acc *account.Account, pri *ecdsa.PrivateKey) *blockchain.FullNode {
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
	node1.AddPeer(node2, node3)
	node2.AddPeer(node1, node3)
	node3.AddPeer(node1, node2)

	node1.Start()
	node2.Start()
	node3.Start()
	time.Sleep(200 * time.Millisecond)
	err := node1.TransferEpfer(*node2.GetAccount().GetAddr(), 50)
	require.NoError(t, err)
	err = node1.TransferEpfer(*node2.GetAccount().GetAddr(), 50)
	require.NoError(t, err)

	time.Sleep(3 * time.Second)
	require.Equal(t, node1.GetChain().Hash(), node2.GetChain().Hash())
	require.Equal(t, node2.GetChain().Hash(), node3.GetChain().Hash())
	require.Equal(t, 3, node1.GetChain().Len())
	fmt.Printf("node1 chain: \n%s", node1.GetChain())

}

func Test_Network_Transfer_Epfer_complex_threenodes(t *testing.T) {
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

	nodeFactory := func(acc *account.Account, pri *ecdsa.PrivateKey) *blockchain.FullNode {
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
	node1.AddPeer(node2, node3)
	node2.AddPeer(node1, node3)
	node3.AddPeer(node1, node2)

	node1.Start()
	node2.Start()
	node3.Start()
	time.Sleep(200 * time.Millisecond)
	go func() {
		err := node1.TransferEpfer(*node2.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
		err = node1.TransferEpfer(*node2.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
	}()

	go func() {
		err := node2.TransferEpfer(*node3.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
		err = node2.TransferEpfer(*node3.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
	}()

	go func() {
		err := node3.TransferEpfer(*node1.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
		err = node3.TransferEpfer(*node1.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
	}()

	time.Sleep(5 * time.Second)
	require.Equal(t, node1.GetChain().Hash(), node2.GetChain().Hash())
	require.Equal(t, node2.GetChain().Hash(), node3.GetChain().Hash())
	require.Equal(t, 7, node1.GetChain().Len())
	fmt.Printf("node1 chain: \n%s", node1.GetChain())

	require.Equal(t, acc1.String(), node1.GetAccount().String())
	require.Equal(t, acc2.String(), node2.GetAccount().String())
	require.Equal(t, acc3.String(), node3.GetAccount().String())

}

func Test_Network_Transfer_Epfer_complex_fivenodes(t *testing.T) {
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
	acc4, pri4 := accountFactory(400, "laptop", 100)
	acc5, pri5 := accountFactory(500, "iphone", 100)

	genesisFactory := func() *block.Block {
		return generateGenesisBlock(storage.CreateSimpleKV, acc1, acc2, acc3, acc4, acc5)
	}

	nodeFactory := func(acc *account.Account, pri *ecdsa.PrivateKey) *blockchain.FullNode {
		sock, err := transp.CreateSocket("127.0.0.1:0")
		require.NoError(t, err)
		//crypto.
		fullNode, _ := z.NewTestFullNode(t,
			z.WithSocket(sock),
			z.WithMessageRegistry(standard.NewRegistry()),
			z.WithPrivateKey(pri),
			z.WithAccount(acc),
			z.WithGenesisBlock(genesisFactory()),
			z.WithAntiEntropy(100*time.Millisecond),
		)
		return fullNode
	}

	node1 := nodeFactory(acc1, pri1)
	node2 := nodeFactory(acc2, pri2)
	node3 := nodeFactory(acc3, pri3)
	node4 := nodeFactory(acc4, pri4)
	node5 := nodeFactory(acc5, pri5)

	node1.AddPeer(node2, node3, node4)
	node2.AddPeer(node1, node3, node5)
	node3.AddPeer(node1, node2)
	node4.AddPeer(node1, node5)
	node5.AddPeer(node4, node2)

	node1.Start()
	node2.Start()
	node3.Start()
	node4.Start()
	node5.Start()
	time.Sleep(200 * time.Millisecond)

	go func() {
		err := node1.TransferEpfer(*node2.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
		err = node1.TransferEpfer(*node2.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
	}()

	go func() {
		err := node2.TransferEpfer(*node3.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
		err = node2.TransferEpfer(*node3.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
	}()

	go func() {
		err := node3.TransferEpfer(*node1.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
		err = node3.TransferEpfer(*node1.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
	}()

	time.Sleep(10 * time.Second)
	require.Equal(t, node1.GetChain().Hash(), node2.GetChain().Hash())
	require.Equal(t, node2.GetChain().Hash(), node3.GetChain().Hash())
	require.Equal(t, node3.GetChain().Hash(), node4.GetChain().Hash())
	require.Equal(t, node4.GetChain().Hash(), node5.GetChain().Hash())

	require.Equal(t, 7, node1.GetChain().Len())
	fmt.Printf("node1 chain: \n%s", node1.GetChain())

	go func() {
		node1.SyncAccount()
	}()
	go func() { node2.SyncAccount() }()
	go func() { node3.SyncAccount() }()

	go func() { node4.SyncAccount() }()
	go func() { node5.SyncAccount() }()
	time.Sleep(2 * time.Second)

	require.Equal(t, acc1.String(), node1.GetAccount().String())
	require.Equal(t, acc2.String(), node2.GetAccount().String())
	require.Equal(t, acc3.String(), node3.GetAccount().String())
	require.Equal(t, acc4.String(), node4.GetAccount().String())
	require.Equal(t, acc5.String(), node5.GetAccount().String())

}

func Test_Network_Transfer_Epfer_complex_attacker_fivenodes(t *testing.T) {
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
	acc4, pri4 := accountFactory(400, "laptop", 100)
	acc5, pri5 := accountFactory(500, "iphone", 100)

	genesisFactory := func() *block.Block {
		return generateGenesisBlock(storage.CreateSimpleKV, acc1, acc2, acc3, acc4, acc5)
	}

	attackerFactory := func(acc *account.Account, pri *ecdsa.PrivateKey) *blockchain.FullNode {
		sock, err := transp.CreateSocket("127.0.0.1:0")
		require.NoError(t, err)
		//crypto.
		fullNode, _ := z.NewTestFullNode(t,
			z.WithSocket(sock),
			z.WithMessageRegistry(standard.NewRegistry()),
			z.WithPrivateKey(pri),
			z.WithAccount(acc),
			z.WithAttacker(),
			z.WithGenesisBlock(genesisFactory()),
			z.WithAntiEntropy(100*time.Millisecond),
		)
		return fullNode
	}

	nodeFactory := func(acc *account.Account, pri *ecdsa.PrivateKey) *blockchain.FullNode {
		sock, err := transp.CreateSocket("127.0.0.1:0")
		require.NoError(t, err)
		//crypto.
		fullNode, _ := z.NewTestFullNode(t,
			z.WithSocket(sock),
			z.WithMessageRegistry(standard.NewRegistry()),
			z.WithPrivateKey(pri),
			z.WithAccount(acc),
			z.WithGenesisBlock(genesisFactory()),
			z.WithAntiEntropy(100*time.Millisecond),
		)
		return fullNode
	}

	node1 := attackerFactory(acc1, pri1)
	node2 := nodeFactory(acc2, pri2)
	node3 := nodeFactory(acc3, pri3)
	node4 := nodeFactory(acc4, pri4)
	node5 := nodeFactory(acc5, pri5)

	node1.AddPeer(node2, node3, node4)
	node2.AddPeer(node1, node3, node5)
	node3.AddPeer(node1, node2)
	node4.AddPeer(node1, node5)
	node5.AddPeer(node4, node2)

	node1.Start()
	node2.Start()
	node3.Start()
	node4.Start()
	node5.Start()
	time.Sleep(200 * time.Millisecond)

	go func() {
		txn := transaction.NewTransaction(0, 10, *node2.GetAccountAddr(), *node1.GetAccountAddr())
		node1.SubmitTxn(txn)

		txn1 := transaction.NewTransaction(1, 10, *node2.GetAccountAddr(), *node1.GetAccountAddr())
		node1.SubmitTxn(txn1)

		txn2 := transaction.NewTransaction(2, 10, *node2.GetAccountAddr(), *node1.GetAccountAddr())
		node1.SubmitTxn(txn2)
		txn3 := transaction.NewTransaction(3, 10, *node2.GetAccountAddr(), *node1.GetAccountAddr())
		node1.SubmitTxn(txn3)
	}()

	go func() {
		err := node2.TransferEpfer(*node3.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
		err = node2.TransferEpfer(*node3.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
	}()

	go func() {
		err := node3.TransferEpfer(*node1.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
		err = node3.TransferEpfer(*node1.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
	}()

	time.Sleep(10 * time.Second)
	//require.Equal(t, node1.GetChain().Hash(), node2.GetChain().Hash())
	require.Equal(t, node2.GetChain().Hash(), node3.GetChain().Hash())
	require.Equal(t, node3.GetChain().Hash(), node4.GetChain().Hash())
	require.Equal(t, node4.GetChain().Hash(), node5.GetChain().Hash())

	require.Equal(t, 5, node2.GetChain().Len())
	fmt.Printf("node1 chain: \n%s", node1.GetChain())
	fmt.Printf("node2 chain: \n%s", node2.GetChain())

	go func() {
		node1.SyncAccount()
	}()
	go func() { node2.SyncAccount() }()
	go func() { node3.SyncAccount() }()

	go func() { node4.SyncAccount() }()
	go func() { node5.SyncAccount() }()
	time.Sleep(2 * time.Second)

	//require.Equal(t, acc1.String(), node1.GetAccount().String())
	require.Equal(t, acc2.String(), node2.GetAccount().String())
	require.Equal(t, acc3.String(), node3.GetAccount().String())
	require.Equal(t, acc4.String(), node4.GetAccount().String())
	require.Equal(t, acc5.String(), node5.GetAccount().String())

}

func Test_Network_Transfer_Epfer_complex_sevennodes(t *testing.T) {
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
	acc4, pri4 := accountFactory(400, "laptop", 100)
	acc5, pri5 := accountFactory(500, "iphone", 100)
	acc6, pri6 := accountFactory(600, "iphone6", 100)
	acc7, pri7 := accountFactory(700, "iphone7", 100)

	genesisFactory := func() *block.Block {
		return generateGenesisBlock(storage.CreateSimpleKV, acc1, acc2, acc3, acc4, acc5, acc6, acc7)
	}

	nodeFactory := func(acc *account.Account, pri *ecdsa.PrivateKey) *blockchain.FullNode {
		sock, err := transp.CreateSocket("127.0.0.1:0")
		require.NoError(t, err)
		//crypto.
		fullNode, _ := z.NewTestFullNode(t,
			z.WithSocket(sock),
			z.WithMessageRegistry(standard.NewRegistry()),
			z.WithPrivateKey(pri),
			z.WithAccount(acc),
			z.WithGenesisBlock(genesisFactory()),
			z.WithAntiEntropy(100*time.Millisecond),
		)
		return fullNode
	}

	node1 := nodeFactory(acc1, pri1)
	node2 := nodeFactory(acc2, pri2)
	node3 := nodeFactory(acc3, pri3)
	node4 := nodeFactory(acc4, pri4)
	node5 := nodeFactory(acc5, pri5)
	node6 := nodeFactory(acc6, pri6)
	node7 := nodeFactory(acc7, pri7)

	node1.AddPeer(node2, node3, node4)
	node2.AddPeer(node1, node3, node5)
	node3.AddPeer(node1, node2)
	node4.AddPeer(node1, node5)
	node5.AddPeer(node4, node2)

	node6.AddPeer(node3, node7)
	node7.AddPeer(node6, node4)

	node1.Start()
	node2.Start()
	node3.Start()
	node4.Start()
	node5.Start()
	node6.Start()
	node7.Start()

	time.Sleep(200 * time.Millisecond)

	go func() {
		err := node1.TransferEpfer(*node2.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
		err = node1.TransferEpfer(*node2.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
		err = node1.TransferEpfer(*node2.GetAccount().GetAddr(), 0)
		require.NoError(t, err)
		err = node1.TransferEpfer(*node2.GetAccount().GetAddr(), 0)
		require.NoError(t, err)
		err = node1.TransferEpfer(*node2.GetAccount().GetAddr(), 0)
		require.NoError(t, err)
	}()

	go func() {
		err := node2.TransferEpfer(*node3.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
		err = node2.TransferEpfer(*node3.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
		err = node2.TransferEpfer(*node3.GetAccount().GetAddr(), 0)
		require.NoError(t, err)
		err = node2.TransferEpfer(*node3.GetAccount().GetAddr(), 0)
		require.NoError(t, err)
		err = node2.TransferEpfer(*node3.GetAccount().GetAddr(), 0)
		require.NoError(t, err)
	}()

	go func() {
		err := node3.TransferEpfer(*node1.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
		err = node3.TransferEpfer(*node1.GetAccount().GetAddr(), 50)
		require.NoError(t, err)
		err = node3.TransferEpfer(*node1.GetAccount().GetAddr(), 0)
		require.NoError(t, err)
		err = node3.TransferEpfer(*node1.GetAccount().GetAddr(), 0)
		require.NoError(t, err)
		err = node3.TransferEpfer(*node1.GetAccount().GetAddr(), 0)
		require.NoError(t, err)
	}()

	time.Sleep(20 * time.Second)
	require.Equal(t, node1.GetChain().Hash(), node2.GetChain().Hash())
	require.Equal(t, node2.GetChain().Hash(), node3.GetChain().Hash())
	require.Equal(t, node3.GetChain().Hash(), node4.GetChain().Hash())
	require.Equal(t, node4.GetChain().Hash(), node5.GetChain().Hash())
	require.Equal(t, node5.GetChain().Hash(), node6.GetChain().Hash())
	require.Equal(t, node6.GetChain().Hash(), node7.GetChain().Hash())

	require.Equal(t, 16, node1.GetChain().Len())
	fmt.Printf("node1 chain: \n%s", node1.GetChain())

	go func() {
		node1.SyncAccount()
	}()
	go func() { node2.SyncAccount() }()
	go func() { node3.SyncAccount() }()

	go func() { node4.SyncAccount() }()
	go func() { node5.SyncAccount() }()
	time.Sleep(2 * time.Second)

	require.Equal(t, acc1.String(), node1.GetAccount().String())
	require.Equal(t, acc2.String(), node2.GetAccount().String())
	require.Equal(t, acc3.String(), node3.GetAccount().String())
	require.Equal(t, acc4.String(), node4.GetAccount().String())
	require.Equal(t, acc5.String(), node5.GetAccount().String())

}

// TODO: ?????????????????????????????????????????? miner ??????????????????. ???????????? miner ?????????????????? PoW ???????????????????????????????????????, ????????????????????????
// TODO: ?????????????????? bot node ????????? transaction ???????????????????????????????????????????????? solid ?????? transaction
// TODO: ?????????????????????, verifyTxn ?????????????????????.
// TODO: ??????????????? catchup ?????????????????????????
