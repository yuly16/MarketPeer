package impl

import (
	"fmt"
	"math/rand"

	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/transport"
)

// Unicast implements peer.Messaging
// send to itself is naturally supported by UDP
func (n *node) strictUnicast(dest string, msg transport.Message) error {
	// assemble a packet
	// relay shall be self
	relay := n.addr()
	header := transport.NewHeader(n.addr(), relay, dest, 0)
	pkt := transport.Packet{Header: &header, Msg: &msg}

	nextDest, err := n.strictSend(pkt)
	if err != nil {
		err = fmt.Errorf("strictUnicast error: %w", err)
		n.Err(err).Send()
	}
	n.Debug().Str("dest", dest).Str("nextDest", nextDest).Str("msg", msg.String()).Str("pkt", pkt.String()).Msg("unicast packet sended")
	return err
}

// blocking send a packet, target is decided by the routing table
// return `nextDest` and error
func (n *node) strictSend(pkt transport.Packet) (string, error) {
	// 1. source should not be changed
	// 2. relay=me
	// 3. dest should de decided by the routing table
	nextDest, err := n.strictNextHop(pkt.Header.Destination)
	if err != nil {
		return nextDest, fmt.Errorf("send error: %w", err)
	}

	// send the pkt
	err = n.sock.Send(nextDest, pkt, 0)
	if err != nil {
		return nextDest, fmt.Errorf("send error: %w", err)
	}
	return nextDest, nil
}

// strictNextHop will not care about neighbors
func (n *node) strictNextHop(dest string) (string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// dest must be known
	nextDest, ok := n.route[dest]
	var err error
	if !ok {
		err = fmt.Errorf("dest=%s is unknown to me=%s", dest, n.addr())
	}
	return nextDest, err
}

// no need to check routing table
func (n *node) forceSend(peer string, pkt transport.Packet) error {
	// send the pkt
	err := n.sock.Send(peer, pkt, 0)
	if err != nil {
		return err
	}
	return nil
}

// blocking send a packet, target is decided by the routing table
// return `nextDest` and error
func (n *node) send(pkt transport.Packet) (string, error) {
	// 1. source should not be changed
	// 2. relay=me
	// 3. dest should de decided by the routing table
	nextDest, err := n.nextHop(pkt.Header.Destination)
	if err != nil {
		return nextDest, fmt.Errorf("send error: %w", err)
	}
	// send the packet
	err = n.forceSend(nextDest, pkt)
	if err != nil {
		return nextDest, fmt.Errorf("send error: %w", err)
	}
	return nextDest, nil
}

func (n *node) nextHop(dest string) (string, error) {
	if n.isNeighbor(dest) {
		return dest, nil
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	// dest must be known
	nextDest, ok := n.route[dest]
	var err error
	if !ok {
		err = fmt.Errorf("dest=%s is unknown to me=%s", dest, n.addr())
	}
	return nextDest, err
}

// AddPeer implements peer.Service
func (n *node) AddPeer(addr ...string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	// we could directly reach the peers
	// NOTE: adding ourselves should have no effects
	n.Info().Strs("peers", addr).Msg("adding peers")
	for i := 0; i < len(addr); i++ {
		n.route[addr[i]] = addr[i]
		if _, ok := n.neighborSet[addr[i]]; !ok {
			n.neighborSet[addr[i]] = struct{}{}
			n.neighbors = append(n.neighbors, addr[i])
		}

	}
	n.Trace().Str("route", n.route.String()).Str("neighbors", fmt.Sprintf("%v", n.neighbors)).Msg("after added")
}

func (n *node) addNeighbor(addr ...string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	// we could directly reach the peers
	// NOTE: adding ourselves should have no effects
	n.Trace().Strs("peers", addr).Msg("adding peers")
	for i := 0; i < len(addr); i++ {
		if _, ok := n.neighborSet[addr[i]]; !ok {
			n.neighborSet[addr[i]] = struct{}{}
			n.neighbors = append(n.neighbors, addr[i])
		}

	}
	n.Trace().Str("neighbors", fmt.Sprintf("%v", n.neighbors)).Msg("after neighbor added")
}

// GetRoutingTable implements peer.Service
func (n *node) GetRoutingTable() peer.RoutingTable {
	n.mu.Lock()
	defer n.mu.Unlock()
	copy := make(map[string]string)
	for k, v := range n.route {
		copy[k] = v
	}
	return copy
}

// SetRoutingEntry implements peer.Service
func (n *node) SetRoutingEntry(origin, relayAddr string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	// If relayAddr is empty then the record must be deleted
	if relayAddr == "" {
		delete(n.route, origin)
	} else {
		// simply overwrite
		n.route[origin] = relayAddr
	}
	n.Info().Str("origin", origin).Str("relay", relayAddr).Msg("set routing entry")
	n.Debug().Str("route", n.route.String()).Msg("routing table after set")
}

// neighbors access shall be protected,
// FIXME: shall it share the same lock with routeMutex?
func (n *node) randNeigh() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.neighbors) == 0 {
		return NONEIGHBOR
	}
	return n.neighbors[rand.Int31n(int32(len(n.neighbors)))]
}

// Q: what if we only have one random neighbor? A: just return it
func (n *node) randNeighExcept(except string) string {
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.neighbors) == 0 {
		return NONEIGHBOR
	}
	if len(n.neighbors) == 1 {
		return n.neighbors[0]
	}
	randN := n.neighbors[rand.Int31n(int32(len(n.neighbors)))]
	for randN == except {
		randN = n.neighbors[rand.Int31n(int32(len(n.neighbors)))]
	}
	return randN
}

func (n *node) hasNeighbor() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.neighbors) != 0
}

func (n *node) isNeighbor(addr string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, nei := range n.neighbors {
		if nei == addr {
			return true
		}
	}
	return false
}

func (n *node) getNeisExcept(excepts ...string) []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	ret := make([]string, 0, len(n.neighbors))
	for _, nei := range n.neighbors {
		filtered := false
		for _, except := range excepts {
			if except == nei {
				filtered = true
				break
			}
		}
		if !filtered {
			ret = append(ret, nei)
		}
	}
	return ret
}

func (n *node) getNeis() []string {
	return n.getNeisExcept(NONEIGHBOR)
}

// func (n *node) neighborLen() int {
// 	n.mu.Lock()
// 	defer n.mu.Unlock()
// 	return len(n.neighbors)
// }
