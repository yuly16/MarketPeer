package integration

import (
	"os"

	"go.dedis.ch/cs438/internal/binnode"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/peer/impl"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/transport/proxy"
	"go.dedis.ch/cs438/transport/udp"
)

var studentFac peer.Factory = impl.NewPeer
var referenceFac peer.Factory

func init() {
	path := os.Getenv("PEER_BIN_PATH")

	if path == "" {
		path = "./node"
	}

	referenceFac = binnode.GetBinnodeFac(path)
}

var udpFac transport.Factory = udp.NewUDP
var proxyFac transport.Factory = proxy.NewProxy
