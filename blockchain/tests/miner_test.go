package tests

import (
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
	fullNode := z.NewTestFullNode(t,
		z.WithSocket(sock),
		z.WithMessageRegistry(standard.NewRegistry()),
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
