package main

import (
	"github.com/ethereum/go-ethereum/crypto"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/registry/standard"
	"go.dedis.ch/cs438/transport/udp"
	"time"
)

func main() {
	transp := udp.NewUDP()
	bitNum := 12
	socket, _ := transp.CreateSocket("127.0.0.1:8000")
	privateKey, _ := crypto.GenerateKey()
	clientNode := *z.NewClient(nil,
		z.WithSocket(socket),
		z.WithMessageRegistry(standard.NewRegistry()),
		z.WithPrivateKey(privateKey),
		z.WithHeartbeat(time.Millisecond*500),
		z.WithChordBits(uint(bitNum)),
		z.WithStabilizeInterval(time.Millisecond*500),
		z.WithFixFingersInterval(time.Millisecond*250))
	clientNode.Start()
	clientNode.AddPeers("127.0.0.1:8000")
	time.Sleep(time.Second * 5)

	time.Sleep(time.Second * 1000)
	clientNode.Stop()
}
