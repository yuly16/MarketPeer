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

type NodeWarp struct {
	node z.TestNode
	id   uint
}

// test 5 test transfer key simple test
func Test_Chord_threePeers_transferKey(t *testing.T) {
	transp := channel.NewTransport()

	nodeNum := 3
	bitNum := 7
	ip2node := map[uint]NodeWarp{}
	nodes := make([]NodeWarp, nodeNum)
	for i := 0; i < nodeNum; i++ {
		nodes[i].node = z.NewTestNode(t, peerFac, transp,
			"127.0.0.1:0",
			z.WithHeartbeat(time.Millisecond*500),
			z.WithChordBits(uint(bitNum)),
			z.WithStabilizeInterval(time.Millisecond*500),
			z.WithFixFingersInterval(time.Millisecond*250))
		nodes[i].id = nodes[i].node.GetChordId()
		ip2node[nodes[i].node.GetChordId()] = nodes[i]
		defer nodes[i].node.Stop()
	}

	// initialize table
	for i := 0; i < nodeNum - 1; i++ {
		nodes[i].node.AddPeer(nodes[i+1].node.GetAddr())
	}
	time.Sleep(6 * time.Second)

	nodes[0].node.Init(nodes[1].node.GetAddr())
	nodes[1].node.Init(nodes[0].node.GetAddr())

	time.Sleep(10 * time.Second)


	expect := map[uint]interface{}{100:"no", 112:"yes"}
	empty := map[uint]interface{}{}
	for k,v := range expect {
		nodes[0].node.PutId(k,v)
	}
	time.Sleep(time.Second * 20)

	require.Equal(t, empty, nodes[0].node.GetChordStorage())
	require.Equal(t, expect, nodes[1].node.GetChordStorage())

	for i := 2; i < nodeNum; i++ {
		require.NoError(t, nodes[i].node.Join(nodes[i-1].node.GetAddr()))
	}
	time.Sleep(time.Second * 20)
	require.Equal(t, nodes[0].node.GetChordStorage(), empty)
	require.Equal(t, nodes[1].node.GetChordStorage(), empty)
	require.Equal(t, nodes[2].node.GetChordStorage(), expect)
}


// test 1: two peers initial a chord system
func Test_Chord_twoPeers_createSystem(t *testing.T) {
	transp := channel.NewTransport()

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0",
		z.WithChordBits(7),
		z.WithStabilizeInterval(2 * time.Second),
		z.WithHeartbeat(time.Millisecond*500))
	defer node1.Stop()
	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0",
		z.WithChordBits(7),
		z.WithStabilizeInterval(2 * time.Second),
		z.WithHeartbeat(time.Millisecond*500))
	defer node2.Stop()
	// initialize routing table
	node1.AddPeer(node2.GetAddr())

	time.Sleep(4 * time.Second)
	node1.Init(node2.GetAddr())
	node2.Init(node1.GetAddr())
	time.Sleep(20 * time.Second)
	require.Equal(t, node1.GetPredecessor(), node2.GetChordId())
	require.Equal(t, node2.GetPredecessor(), node1.GetChordId())
	require.Equal(t, node1.GetSuccessor(), node2.GetChordId())
	require.Equal(t, node2.GetSuccessor(), node1.GetChordId())
}


// test 2: 6 node create a system
func Test_Chord_sixPeers_createSystem(t *testing.T) {
	//transp := channel.NewTransport()
	transp := udp.NewUDP()
	nodeNum := 6
	bitNum := 12

	nodes := make([]NodeWarp, nodeNum)
	for i := 0; i < nodeNum; i++ {
		nodes[i].node = z.NewTestNode(t, peerFac, transp,
			"127.0.0.1:0",
			z.WithHeartbeat(time.Millisecond*500),
			z.WithChordBits(uint(bitNum)),
			z.WithStabilizeInterval(time.Millisecond*500),
			z.WithFixFingersInterval(time.Millisecond*250))
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
	time.Sleep(120 * time.Second)
	fmt.Println("chord ends")
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].id < nodes[j].id
	})
	for i := 0; i < nodeNum; i++ {
		nodes[i].node.PrintInfo()
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
	for i := 0; i < nodeNum; i++ {
		fingerTable := nodes[i].node.GetFingerTable()
		chordId := nodes[i].id
		for j := 0; j < bitNum; j++ {
			biasId := (chordId + 1 << j) % (1 << bitNum)
			var expect uint
			for k := 0; k < nodeNum; k++ {
				if betweenRightInclude(biasId, nodes[k].id, nodes[(k+1) % nodeNum].id) {
					expect = nodes[(k+1) % nodeNum].id
					break
				}
			}
			require.Equal(t, expect, fingerTable[j])
		}
	}
}


// test 3: 14 node create a system
func Test_Chord_Peers_createSystem(t *testing.T) {
	//transp := channel.NewTransport()
	transp := udp.NewUDP()
	nodeNum := 10
	bitNum := 12

	nodes := make([]NodeWarp, nodeNum)
	for i := 0; i < nodeNum; i++ {
		nodes[i].node = z.NewTestNode(t, peerFac, transp,
			"127.0.0.1:0",
			z.WithHeartbeat(time.Millisecond*500),
			z.WithChordBits(uint(bitNum)),
			z.WithStabilizeInterval(time.Millisecond*500),
			z.WithFixFingersInterval(time.Millisecond*250))
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
	time.Sleep(120 * time.Second)
	fmt.Println("chord ends")
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].id < nodes[j].id
	})
	for i := 0; i < nodeNum; i++ {
		nodes[i].node.PrintInfo()
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
	for i := 0; i < nodeNum; i++ {
		fingerTable := nodes[i].node.GetFingerTable()
		chordId := nodes[i].id
		for j := 0; j < bitNum; j++ {
			biasId := (chordId + 1 << j) % (1 << bitNum)
			var expect uint
			for k := 0; k < nodeNum; k++ {
				if betweenRightInclude(biasId, nodes[k].id, nodes[(k+1) % nodeNum].id) {
					expect = nodes[(k+1) % nodeNum].id
					break
				}
			}
			require.Equal(t, expect, fingerTable[j])
		}
	}
	fmt.Println("dsfghg")
}


// test 4 if 10 peers can find the correct ip address
func Test_Chord_Peers_lookup(t *testing.T) {
	//transp := channel.NewTransport()
	transp := udp.NewUDP()
	nodeNum := 7
	bitNum := 12
	ip2node := map[uint]NodeWarp{}
	nodes := make([]NodeWarp, nodeNum)
	for i := 0; i < nodeNum; i++ {
		nodes[i].node = z.NewTestNode(t, peerFac, transp,
			"127.0.0.1:0",
			z.WithHeartbeat(time.Millisecond*500),
			z.WithChordBits(uint(bitNum)),
			z.WithStabilizeInterval(time.Millisecond*500),
			z.WithFixFingersInterval(time.Millisecond*250))
		nodes[i].id = nodes[i].node.GetChordId()
		ip2node[nodes[i].node.GetChordId()] = nodes[i]
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
	time.Sleep(120 * time.Second)
	fmt.Println("chord ends")
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].id < nodes[j].id
	})
	for i := 0; i < nodeNum; i++ {
		nodes[i].node.PrintInfo()
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
	for i := 0; i < nodeNum; i++ {
		fingerTable := nodes[i].node.GetFingerTable()
		chordId := nodes[i].id
		for j := 0; j < bitNum; j++ {
			biasId := (chordId + 1 << j) % (1 << bitNum)
			var expect uint
			for k := 0; k < nodeNum; k++ {
				if betweenRightInclude(biasId, nodes[k].id, nodes[(k+1) % nodeNum].id) {
					expect = nodes[(k+1) % nodeNum].id
					break
				}
			}
			require.Equal(t, expect, fingerTable[j])
		}
	}

	fmt.Println("putting...")
	for i := 0; i < 1 << bitNum; i = i + 1 << 3 {
		dest, err := nodes[0].node.LookupHashId(uint(i))
		var expect uint

		require.NoError(t, err)
		require.NotEqual(t, "", dest)

		for k := 0; k < nodeNum; k++ {
			if betweenRightInclude(uint(i), nodes[k].id, nodes[(k+1) % nodeNum].id) {
				expect = nodes[(k+1) % nodeNum].id
				break
			}
		}
		require.Equal(t, expect, dest)
		ip2node[dest].node.PutId(uint(i), uint(i))
	}
	fmt.Println("getting...")
	for i := 0; i < 1 << bitNum; i = i + 1 << 3 {
		dest, err := nodes[0].node.LookupHashId(uint(i))
		if err != nil {
			require.Error(t, err)
		}
		res, exists := ip2node[dest].node.GetId(uint(i))
		require.Equal(t, exists, true)
		require.Equal(t, uint(i), res)
	}
}




// test 6 test transfer key big scenario
func Test_Chord_Peers_transferKey(t *testing.T) {
	transp := channel.NewTransport()
	// this test is forbidden to use udp!
	//transp := udp.NewUDP()
	nodeNum := 7
	bitNum := 12
	ip2node := map[uint]NodeWarp{}
	nodes := make([]NodeWarp, nodeNum)
	for i := 0; i < nodeNum; i++ {
		nodes[i].node = z.NewTestNode(t, peerFac, transp,
			"127.0.0.1:0",
			z.WithHeartbeat(time.Millisecond*500),
			z.WithChordBits(uint(bitNum)),
			z.WithStabilizeInterval(time.Millisecond*500),
			z.WithFixFingersInterval(time.Millisecond*250))
		nodes[i].id = nodes[i].node.GetChordId()
		ip2node[nodes[i].node.GetChordId()] = nodes[i]
		defer nodes[i].node.Stop()
	}

	// initialize table
	for i := 0; i < nodeNum - 1; i++ {
		nodes[i].node.AddPeer(nodes[i+1].node.GetAddr())
	}
	time.Sleep(6 * time.Second)

	nodes[0].node.Init(nodes[1].node.GetAddr())
	nodes[1].node.Init(nodes[0].node.GetAddr())

	time.Sleep(10 * time.Second)

	fmt.Println("push kv...")

	for i := 0; i < 1 << bitNum; i = i + 1 << 3 {
		nodes[0].node.PutId(uint(i), i)
	}


	time.Sleep(time.Second * 5)
	for i := 0; i < nodeNum; i++ {
		fmt.Println(nodes[i].node.GetChordStorage())
	}
	fmt.Println("joining new node...")
	for i := 2; i < nodeNum; i++ {
		require.NoError(t, nodes[i].node.Join(nodes[i-1].node.GetAddr()))
	}
	time.Sleep(time.Second * 120)
	fmt.Println("getting...")
	for i := 0; i < 1 << bitNum; i = i + 1 << 3 {
		dest, err := nodes[0].node.LookupHashId(uint(i))
		if err != nil {
			require.Error(t, err)
		}
		res, exists := ip2node[dest].node.GetId(uint(i))
		require.Equal(t, exists, true)

		switch res.(type) {
		case int:
			require.Equal(t, i, res)
		case float64:
			require.Equal(t, i, int(res.(float64)))
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