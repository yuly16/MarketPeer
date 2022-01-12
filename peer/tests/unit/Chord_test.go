package unit

import (
	"fmt"
	"github.com/stretchr/testify/require"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/transport/channel"
	"go.dedis.ch/cs438/transport/udp"
	"sort"
	"testing"
	"time"
)

// test 1
func Test_Chord_twoPeers_createSystem(t *testing.T) {
	transp := channel.NewTransport()

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithChordBits(7), z.WithStabilizeInterval(2 * time.Second))
	defer node1.Stop()
	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithChordBits(7), z.WithStabilizeInterval(2 * time.Second))
	defer node2.Stop()
	// initialize routing table
	node1.AddPeer(node2.GetAddr())

	time.Sleep(1 * time.Second)
	node1.Join(node2.GetAddr())
	time.Sleep(6 * time.Second)
	node1.Test()
	node2.Test()
	fmt.Println("dsfghg")
}

// test 2
func Test_Chord_threePeers_createSystem(t *testing.T) {
	transp := channel.NewTransport()

	node1 := z.NewTestNode(t, peerFac, transp,
		"127.0.0.1:0",
		z.WithHeartbeat(time.Millisecond*500),
		z.WithChordBits(7),
		z.WithStabilizeInterval(1 * time.Second))
	defer node1.Stop()
	node2 := z.NewTestNode(t, peerFac, transp,
		"127.0.0.1:0",
		z.WithHeartbeat(time.Millisecond*500),
		z.WithChordBits(7),
		z.WithStabilizeInterval(1 * time.Second))
	defer node2.Stop()
	node3 := z.NewTestNode(t, peerFac, transp,
		"127.0.0.1:0",
		z.WithHeartbeat(time.Millisecond*500),
		z.WithChordBits(7),
		z.WithStabilizeInterval(1 * time.Second))
	defer node3.Stop()
	// initialize routing table
	node1.AddPeer(node2.GetAddr())
	node2.AddPeer(node3.GetAddr())
	time.Sleep(3 * time.Second)
	node1.Init(node2.GetAddr())
	node2.Init(node1.GetAddr())
	node3.Join(node1.GetAddr())
	time.Sleep(5 * time.Second)
	node1.Test()
	node2.Test()
	node3.Test()
	fmt.Println("dsfghg")
}

// test 3
func Test_Chord_fourPeers_createSystem(t *testing.T) {
	transp := udp.NewUDP()

	node1 := z.NewTestNode(t, peerFac, transp,
		"127.0.0.1:0",
		z.WithHeartbeat(time.Millisecond*500),
		z.WithChordBits(7),
		z.WithStabilizeInterval(time.Millisecond*500),
		z.WithFixFingersInterval(time.Millisecond*500))
	defer node1.Stop()
	node2 := z.NewTestNode(t, peerFac, transp,
		"127.0.0.1:0",
		z.WithHeartbeat(time.Millisecond*500),
		z.WithChordBits(7),
		z.WithStabilizeInterval(time.Millisecond*500),
		z.WithFixFingersInterval(time.Millisecond*500))
	defer node2.Stop()
	node3 := z.NewTestNode(t, peerFac, transp,
		"127.0.0.1:0",
		z.WithHeartbeat(time.Millisecond*500),
		z.WithChordBits(7),
		z.WithStabilizeInterval(time.Millisecond*500),
		z.WithFixFingersInterval(time.Millisecond*500))
	defer node3.Stop()
	node4 := z.NewTestNode(t, peerFac, transp,
		"127.0.0.1:0",
		z.WithHeartbeat(time.Millisecond*500),
		z.WithChordBits(7),
		z.WithStabilizeInterval(time.Millisecond*500),
		z.WithFixFingersInterval(time.Millisecond*500))
	defer node4.Stop()
	node5 := z.NewTestNode(t, peerFac, transp,
		"127.0.0.1:0",
		z.WithHeartbeat(time.Millisecond*500),
		z.WithChordBits(7),
		z.WithStabilizeInterval(time.Millisecond*500),
		z.WithFixFingersInterval(time.Millisecond*500))
	defer node5.Stop()
	node6 := z.NewTestNode(t, peerFac, transp,
		"127.0.0.1:0",
		z.WithHeartbeat(time.Millisecond*500),
		z.WithChordBits(7),
		z.WithStabilizeInterval(time.Millisecond*500),
		z.WithFixFingersInterval(time.Millisecond*500))
	defer node6.Stop()

	// initialize routing table
	node1.AddPeer(node2.GetAddr())
	node2.AddPeer(node3.GetAddr())
	node3.AddPeer(node4.GetAddr())
	node4.AddPeer(node5.GetAddr())
	node5.AddPeer(node6.GetAddr())
	time.Sleep(6 * time.Second)
	node2.Init(node1.GetAddr())
	node1.Init(node2.GetAddr())
	node3.Join(node1.GetAddr())
	node4.Join(node1.GetAddr())
	node5.Join(node1.GetAddr())
	node6.Join(node1.GetAddr())
	time.Sleep(20 * time.Second)

	node1.Test()
	node2.Test()
	node3.Test()
	node4.Test()
	node5.Test()
	node6.Test()
	fmt.Println("dsfghg")
}


// test 4
type NodeWarp struct {
	node z.TestNode
	id   uint
}
func Test_Chord_tenPeers_createSystem(t *testing.T) {
	//transp := channel.NewTransport()
	transp := udp.NewUDP()
	nodeNum := 16
	bitNum := 12

	nodes := make([]NodeWarp, nodeNum)
	for i := 0; i < nodeNum; i++ {
		nodes[i].node = z.NewTestNode(t, peerFac, transp,
			"127.0.0.1:0",
			z.WithHeartbeat(time.Millisecond*500),
			z.WithChordBits(uint(bitNum)),
			z.WithStabilizeInterval(time.Millisecond*500),
			z.WithFixFingersInterval(time.Millisecond*500))
		nodes[i].id = nodes[i].node.GetChordId()
		defer nodes[i].node.Stop()
	}

	// initialize table
	for i := 0; i < nodeNum - 1; i++ {
		nodes[i].node.AddPeer(nodes[i+1].node.GetAddr())
	}
	time.Sleep(6 * time.Second)

	nodes[0].node.Init(nodes[1].node.GetAddr())
	nodes[1].node.Init(nodes[0].node.GetAddr())

	for i := 2; i < nodeNum; i++ {
		require.NoError(t, nodes[i].node.Join(nodes[i-1].node.GetAddr()))
	}
	fmt.Println("chord starts...")
	time.Sleep(30 * time.Second)
	fmt.Println("chord ends")
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].id < nodes[j].id
	})
	for i := 0; i < nodeNum; i++ {
		nodes[i].node.Test()
	}
	// check predecessor and successor
	for i := 0; i < nodeNum; i++ {
		successor := nodes[i].node.GetSuccessor()
		expect := nodes[(i + 1) % nodeNum].id
		require.Equal(t, expect, successor)
	}
	for i := 0; i < nodeNum; i++ {
		predecessor := nodes[i].node.GetPredecessor()
		expect := nodes[(i + nodeNum - 1) % nodeNum].id
		require.Equal(t, expect, predecessor)
	}

	// check fingerTable
	//for i := 0; i < nodeNum; i++ {
	//	fingerTable := nodes[i].node.GetFingerTable()
	//	chordId := nodes[i].id
	//	for j := 0; j < bitNum; j++ {
	//		biasId := (chordId + 1 << j) % (1 << bitNum)
	//		var expect uint
	//		for k := 0; k < nodeNum; k++ {
	//			if betweenRightInclude(biasId, nodes[k].id, nodes[(k+1) % nodeNum].id) {
	//				expect = nodes[(k+1) % nodeNum].id
	//				break
	//			}
	//		}
	//		require.Equal(t, expect, fingerTable[j])
	//	}
	//}
	fmt.Println("dsfghg")
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