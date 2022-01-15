package tests

import (
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/client/client"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/registry/standard"
	"go.dedis.ch/cs438/transport/channel"
	"sort"
	"testing"
	"time"
)




func TestSimpleScenario(t *testing.T) {
	//var kvFactory storage.KVFactory = storage.CreateSimpleKV
	transp := channel.NewTransport()
	nodeNum := 3
	bitNum := 7
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
		require.Equal(t, expect, successor)
	}
	for i := 0; i < nodeNum; i++ {
		predecessor := nodes[i].ChordNode.GetPredecessor()
		expect := nodes[(i + nodeNum - 1) % nodeNum].ChordNode.GetChordId()
		require.Equal(t, expect, predecessor)
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