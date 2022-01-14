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

func TestBuildFullNode(t *testing.T) {
	// init some network nodes, which have some balance
	transp := channel.NewTransport()
	sock, err := transp.CreateSocket("127.0.0.1:0")
	require.NoError(t, err)
	privateKey, err := rsa.GenerateKey(rand.Reader, 16)
	require.NoError(t, err)
	publicKey := privateKey.PublicKey
	fullNode, _ := z.NewTestFullNode(t,
		z.WithSocket(sock),
		z.WithMessageRegistry(standard.NewRegistry()),
		z.WithPrivateKey(*privateKey),
		z.WithPublicKey(publicKey),
	)
	fullNode.Start()

	time.Sleep(1 * time.Second)
	fullNode.Stop()

}

func TestNetworkInit(t *testing.T) {

}

func TestOneConsensus(t *testing.T) {
	// a wallet
}
