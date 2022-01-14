package tests

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/registry/standard"
	"go.dedis.ch/cs438/transport/channel"

	z "go.dedis.ch/cs438/internal/testing"
)

func TestTxnSubmit(t *testing.T) {
	// init some network nodes, which have some balance
	transp := channel.NewTransport()
	sock1, err := transp.CreateSocket("127.0.0.1:0")

	require.NoError(t, err)
	privateKey, err := rsa.GenerateKey(rand.Reader, 16)
	require.NoError(t, err)
	publicKey := privateKey.PublicKey

	fullNode1, messager1 := z.NewTestFullNode(t,
		z.WithSocket(sock1),
		z.WithMessageRegistry(standard.NewRegistry()),
		z.WithPrivateKey(*privateKey),
		z.WithPublicKey(publicKey),
	)
	fullNode1.Start()
	defer fullNode1.Stop()

	sock2, err := transp.CreateSocket("127.0.0.1:0")
	fullNode2, _ := z.NewTestFullNode(t,
		z.WithSocket(sock2),
		z.WithMessageRegistry(standard.NewRegistry()),
		z.WithPrivateKey(*privateKey),
		z.WithPublicKey(publicKey),
	)
	fullNode2.Start()
	defer fullNode2.Stop()

	sock3, err := transp.CreateSocket("127.0.0.1:0")
	fullNode3, _ := z.NewTestFullNode(t,
		z.WithSocket(sock3),
		z.WithMessageRegistry(standard.NewRegistry()),
		z.WithHeartbeat(time.Microsecond * 500),
		z.WithPrivateKey(*privateKey),
		z.WithPublicKey(publicKey),
	)
	fullNode3.Start()
	defer fullNode3.Stop()

	messager1.AddPeer(sock2.GetAddress())
	messager1.AddPeer(sock3.GetAddress())
	time.Sleep(4 * time.Second)

	fullNode1.Test_submitTxn()
	time.Sleep(1000 * time.Second)
}

