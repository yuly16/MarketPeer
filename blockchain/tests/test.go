package tests

import (
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/blockchain"
	"go.dedis.ch/cs438/blockchain/messaging"
	"go.dedis.ch/cs438/blockchain/storage"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/registry/standard"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/transport/channel"
	"testing"
)

func CreateFullNode(t *testing.T, transport transport.Transport) (*blockchain.FullNode, messaging.Messager) {
	sock, err := transport.CreateSocket("127.0.0.1:0")
	require.NoError(t, err)
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	fullNode, messager := z.NewTestFullNode(t,
		z.WithSocket(sock),
		z.WithMessageRegistry(standard.NewRegistry()),
		z.WithPrivateKey(privateKey),
	)
	fullNode.Start()
	defer fullNode.Stop()
	return fullNode, messager
}


func TestSimpleScenario(t *testing.T) {
	var kvFactory storage.KVFactory = storage.CreateSimpleKV
	transp := channel.NewTransport()
	nodeNum := 4
	nodes := make([]blockchain.FullNode, nodeNum)
	for i := 0; i < nodeNum; i++ {
		fmt.Println(nodes)
		fmt.Println(kvFactory)
	}
	CreateFullNode(t, transp)





}