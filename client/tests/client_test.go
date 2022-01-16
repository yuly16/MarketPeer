package tests

import (
	"crypto/ecdsa"
	"crypto/sha1"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/blockchain/transaction"
	"go.dedis.ch/cs438/client/client"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/registry/standard"
	"go.dedis.ch/cs438/transport/channel"
	"math/big"
	"sort"
	"testing"
	"time"
)

// based on init accounts info, generate the genesis block
func generateGenesisBlock(kvFactory storage.KVFactory, accounts ...*account.Account) *block.Block {
	bb := block.NewBlockBuilder(kvFactory).
		SetParentHash(block.DUMMY_PARENT_HASH).
		SetNonce(0).
		SetNumber(0).
		SetDifficulty(2).
		SetBeneficiary(*account.NewAddress([8]byte{}))
	for _, acc := range accounts {
		bb.SetAddrState(acc.GetAddr(), acc.GetState())
	}
	b := bb.Build()
	return b
}


func Test_Client_SimpleScenario(t *testing.T) {
	//var kvFactory storage.KVFactory = storage.CreateSimpleKV
	transp := channel.NewTransport()
	nodeNum := 10
	bitNum := 12
	nodes := make([]client.Client, nodeNum)
	for i := 0; i < nodeNum; i++ {
		sock, err := transp.CreateSocket("127.0.0.1:0")
		require.NoError(t, err)
		privateKey, err := crypto.GenerateKey()
		require.NoError(t, err)
		nodes[i] = *z.NewClient(t,
			z.WithSocket(sock),
			z.WithMessageRegistry(standard.NewRegistry()),
			z.WithPrivateKey(privateKey),
			z.WithHeartbeat(time.Millisecond*500),
			z.WithChordBits(uint(bitNum)),
			z.WithStabilizeInterval(time.Millisecond*500),
			z.WithFixFingersInterval(time.Millisecond*250))
		nodes[i].Start()
		defer nodes[i].Stop()
	}

	for i := 1; i < nodeNum; i++ {
		nodes[i].AddPeers(nodes[i-1].Address)
	}
	time.Sleep(time.Second * 7)

	nodes[0].ChordNode.Init(nodes[1].Address)
	nodes[1].ChordNode.Init(nodes[0].Address)

	for i := 2; i < nodeNum; i++ {
		fmt.Println(i)
		require.NoError(t, nodes[i].ChordNode.Join(nodes[i-1].Address))
	}
	fmt.Println("chord starts...")
	time.Sleep(120 * time.Second)
	fmt.Println("chord ends")
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ChordNode.GetChordId() < nodes[j].ChordNode.GetChordId()
	})

	// check predecessor and successor
	for i := 0; i < nodeNum; i++ {
		successor := nodes[i].ChordNode.GetSuccessor()
		expect := nodes[(i + 1) % nodeNum].ChordNode.GetChordId()
		require.Equal(t, expect, HashKey(successor, uint(bitNum)))
	}
	for i := 0; i < nodeNum; i++ {
		predecessor := nodes[i].ChordNode.GetPredecessor()
		expect := nodes[(i + nodeNum - 1) % nodeNum].ChordNode.GetChordId()
		require.Equal(t, expect, HashKey(predecessor, uint(bitNum)))
	}

	// check fingerTable
	for i := 0; i < nodeNum; i++ {
		fingerTable := nodes[i].ChordNode.GetFingerTable()
		chordId := nodes[i].ChordNode.GetChordId()
		for j := 0; j < bitNum; j++ {
			biasId := (chordId + 1 << j) % (1 << bitNum)
			var expect uint
			for k := 0; k < nodeNum; k++ {
				if betweenRightInclude(biasId, nodes[k].ChordNode.GetChordId(), nodes[(k+1) % nodeNum].ChordNode.GetChordId()) {
					expect = nodes[(k+1) % nodeNum].ChordNode.GetChordId()
					break
				}
			}
			require.Equal(t, expect, fingerTable[j])
		}
	}
}

func Test_Client_ProductStorage(t *testing.T) {
	//var kvFactory storage.KVFactory = storage.CreateSimpleKV
	transp := channel.NewTransport()
	nodeNum := 5
	bitNum := 12
	nodes := make([]client.Client, nodeNum)
	for i := 0; i < nodeNum; i++ {
		sock, err := transp.CreateSocket("127.0.0.1:0")
		require.NoError(t, err)
		privateKey, err := crypto.GenerateKey()
		require.NoError(t, err)
		nodes[i] = *z.NewClient(t,
			z.WithSocket(sock),
			z.WithMessageRegistry(standard.NewRegistry()),
			z.WithPrivateKey(privateKey),
			z.WithHeartbeat(time.Millisecond*500),
			z.WithChordBits(uint(bitNum)),
			z.WithStabilizeInterval(time.Millisecond*500),
			z.WithFixFingersInterval(time.Millisecond*250))
		nodes[i].Start()
		defer nodes[i].Stop()
	}

	for i := 1; i < nodeNum; i++ {
		nodes[i].AddPeers(nodes[i-1].Address)
	}
	time.Sleep(time.Second * 7)

	nodes[0].ChordNode.Init(nodes[1].Address)
	nodes[1].ChordNode.Init(nodes[0].Address)

	for i := 2; i < nodeNum; i++ {
		fmt.Println(i)
		require.NoError(t, nodes[i].ChordNode.Join(nodes[i-1].Address))
	}
	fmt.Println("chord starts...")
	time.Sleep(120 * time.Second)
	fmt.Println("chord ends")


	orange := client.Product{
		Name: "orange",
		Owner: nodes[0].Address,
		Stock: 1000,
	}
	apple := client.Product{
		Name: "apple",
		Owner: nodes[1].Address,
		Stock: 400,
	}
	banana := client.Product{
		Name: "banana",
		Owner: nodes[0].Address,
		Stock: 600,
	}

	orange_key := HashKey(orange.Name, uint(bitNum))
	apple_key := HashKey(apple.Name, uint(bitNum))
	banana_key := HashKey(banana.Name, uint(bitNum))
	clientNode := nodes[0]
	fmt.Println("client stores a product")
	err := clientNode.StoreProduct(orange_key, orange)
	require.NoError(t, err)
	err1 := clientNode.StoreProduct(apple_key, apple)
	require.NoError(t, err)
	err2 := clientNode.StoreProduct(banana_key, banana)
	require.NoError(t, err)
	require.NoError(t, err1)
	require.NoError(t, err2)

	time.Sleep(time.Second * 3)
	fmt.Println("client reads a product")
	actualOrange, ok := clientNode.ReadProduct(orange_key)
	require.Equal(t, true, ok)
	require.Equal(t, orange, actualOrange)

	actualApple, ok := clientNode.ReadProduct(apple_key)
	require.Equal(t, true, ok)
	require.Equal(t, apple, actualApple)

	actualBanana, ok := clientNode.ReadProduct(banana_key)
	require.Equal(t, true, ok)
	require.Equal(t, banana, actualBanana)
}

func Test_Client_SubmitTransaction(t *testing.T) {
	transp := channel.NewTransport()
	bitNum := 12
	// create account
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

	nodeFactory := func(acc *account.Account, pri *ecdsa.PrivateKey) *client.Client {
		sock, err := transp.CreateSocket("127.0.0.1:0")
		require.NoError(t, err)
		//crypto.
		node := *z.NewClient(t,
			z.WithSocket(sock),
			z.WithMessageRegistry(standard.NewRegistry()),
			z.WithPrivateKey(pri),
			z.WithAccount(acc),
			z.WithGenesisBlock(genesisFactory()),
			z.WithHeartbeat(time.Millisecond*500),
			z.WithChordBits(uint(bitNum)),
			z.WithStabilizeInterval(time.Millisecond*500),
			z.WithFixFingersInterval(time.Millisecond*250))
		node.Start()
		return &node
	}

	node1 := nodeFactory(acc1, pri1)
	node2 := nodeFactory(acc2, pri2)
	node3 := nodeFactory(acc3, pri3)

	node1.AddPeers(node2.Address)
	node1.AddPeers(node3.Address)

	time.Sleep(time.Second * 5)

	txn1 := transaction.NewTransaction(0, 5,
		*node1.BlockChainFullNode.GetAccountAddr(),
		*node2.BlockChainFullNode.GetAccountAddr())
	txn2 := transaction.NewTransaction(0, 5,
		*node2.BlockChainFullNode.GetAccountAddr(),
		*node3.BlockChainFullNode.GetAccountAddr())
	txn3 := transaction.NewTransaction(0, 5,
		*node3.BlockChainFullNode.GetAccountAddr(),
		*node1.BlockChainFullNode.GetAccountAddr())

	node1.BlockChainFullNode.SubmitTxn(txn1)
	node2.BlockChainFullNode.SubmitTxn(txn2)
	node3.BlockChainFullNode.SubmitTxn(txn3)

	time.Sleep(3 * time.Second)
	fmt.Printf("node1 chain: \n%s", node1.BlockChainFullNode.GetChain())
	fmt.Printf("node2 chain: \n%s", node2.BlockChainFullNode.GetChain())
	fmt.Printf("node3 chain: \n%s", node3.BlockChainFullNode.GetChain())

	require.Equal(t, node1.BlockChainFullNode.GetChain().Hash(),
		node2.BlockChainFullNode.GetChain().Hash())
	require.Equal(t, node2.BlockChainFullNode.GetChain().Hash(),
		node3.BlockChainFullNode.GetChain().Hash())
	go func() {
		node1.BlockChainFullNode.SyncAccount()
	}()
	go func() { node2.BlockChainFullNode.SyncAccount() }()
	go func() { node3.BlockChainFullNode.SyncAccount() }()
	time.Sleep(3 * time.Second)
	require.Equal(t, acc1.String(), node1.BlockChainFullNode.GetAccount().String())
	require.Equal(t, acc2.String(), node2.BlockChainFullNode.GetAccount().String())
	require.Equal(t, acc3.String(), node3.BlockChainFullNode.GetAccount().String())
	node1.Stop()
	node2.Stop()
	node3.Stop()
}


func betweenRightInclude(id uint, left uint, right uint) bool {
	return between(id, left, right) || id == right
}

func between(id uint, left uint, right uint) bool {
	if right > left {
		return id > left && id < right
	} else if right < left {
		return id < right || id > left
	} else {
		return false
	}
}

func HashKey(key string, ChordBits uint) uint {
	h := sha1.New()
	if _, err := h.Write([]byte(key)); err != nil {
		return 0
	}
	val := h.Sum(nil)
	valInt := (&big.Int{}).SetBytes(val)

	return uint(valInt.Uint64()) % (1 << ChordBits)
}