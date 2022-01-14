package tests

import (
	"github.com/ethereum/go-ethereum/crypto"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/registry/standard"
	"go.dedis.ch/cs438/transport/channel"

	z "go.dedis.ch/cs438/internal/testing"
)

func TestTxnSubmitAndVerify(t *testing.T) {
	// init some network nodes, which have some balance
	transp := channel.NewTransport()
	sock1, err := transp.CreateSocket("127.0.0.1:0")

	require.NoError(t, err)
	privateKey1, err := crypto.GenerateKey()
	require.NoError(t, err)


	fullNode1, messager1 := z.NewTestFullNode(t,
		z.WithSocket(sock1),
		z.WithMessageRegistry(standard.NewRegistry()),
		z.WithPrivateKey(privateKey1),
	)
	fullNode1.Start()
	defer fullNode1.Stop()

	require.NoError(t, err)
	privateKey2, err := crypto.GenerateKey()
	require.NoError(t, err)

	sock2, err := transp.CreateSocket("127.0.0.1:0")
	fullNode2, _ := z.NewTestFullNode(t,
		z.WithSocket(sock2),
		z.WithMessageRegistry(standard.NewRegistry()),
		z.WithPrivateKey(privateKey2),
	)
	fullNode2.Start()
	defer fullNode2.Stop()

	
	require.NoError(t, err)
	privateKey3, err := crypto.GenerateKey()
	require.NoError(t, err)

	sock3, err := transp.CreateSocket("127.0.0.1:0")
	fullNode3, _ := z.NewTestFullNode(t,
		z.WithSocket(sock3),
		z.WithMessageRegistry(standard.NewRegistry()),
		z.WithHeartbeat(time.Microsecond * 500),
		z.WithPrivateKey(privateKey3),
	)
	fullNode3.Start()
	defer fullNode3.Stop()

	messager1.AddPeer(sock2.GetAddress())
	messager1.AddPeer(sock3.GetAddress())
	time.Sleep(4 * time.Second)

	fullNode1.Test_submitTxn()
	time.Sleep(10 * time.Second)
}

