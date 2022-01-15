package tests

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/blockchain"
	"go.dedis.ch/cs438/blockchain/messaging"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/registry/standard"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/transport/channel"
	"testing"
)

func CreateFullNode(t *testing.T, transport transport.Transport) (*blockchain.FullNode,
	messaging.Messager, transport.ClosableSocket) {
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
	return fullNode, messager, sock
}

type FullNodeWrapper struct {
	FullNode *blockchain.FullNode
	Messager messaging.Messager
	Socket   transport.ClosableSocket
}

func TestSimpleScenario(t *testing.T) {
	//var kvFactory storage.KVFactory = storage.CreateSimpleKV
	transp := channel.NewTransport()
	nodeNum := 4
	nodes := make([]FullNodeWrapper, nodeNum)
	for i := 0; i < nodeNum; i++ {
		fullNode, messager, sock := CreateFullNode(t, transp)
		nodes[i].FullNode = fullNode
		nodes[i].Messager = messager
		nodes[i].Socket = sock
	}
	//
	//// Initialize routing table
	//for i := 1; i < nodeNum; i++ {
	//	nodes[i].Messager.AddPeer(nodes[i-1].Socket.GetAddress())
	//}
	//time.Sleep(time.Second * 4)
	//// Initialize chord configuration
	//require.LessOrEqual(t, 2, nodeNum)
	//nodes[0].FullNode.Init(nodes[1].node.GetAddr())
	//nodes[1].node.Init(nodes[0].node.GetAddr())
	//
	//for i := 2; i < nodeNum; i++ {
	//	fmt.Println(i)
	//	require.NoError(t, nodes[i].node.Join(nodes[i-1].node.GetAddr()))
	//}
	//time.Sleep(time.Second * 30)
	//




}