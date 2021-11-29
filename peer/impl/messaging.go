package impl

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/registry"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

// implements peer.Messaging
type Messager struct {
	sock transport.Socket
	zerolog.Logger
	msgRegistry registry.Registry
	conf        peer.Configuration

	// broadcast
	acuMu      sync.Mutex
	ackFutures map[string]chan int

	mu          sync.Mutex // protect access to `route` and `neighbors`, for example, listenDaemon and Unicast, redirect will access it
	neighbors   []string   // it will only grow, since no node will leave the network in assumption
	neighborSet map[string]struct{}
	route       peer.RoutingTable

	seqMu  sync.Mutex               // protect seqs and rumors
	seqs   map[string]uint          // rumor seq of other nodes, here nodes are not necessarily the neighbors, since rumor corresponds to one origin
	rumors map[string][]types.Rumor // key: node_addr value: rumors in increasing order of seq
}

func NewMessager(conf peer.Configuration) *Messager {
	mess := &Messager{sock: conf.Socket, msgRegistry: conf.MessageRegistry, conf: conf}
	mess.route = peer.RoutingTable{mess.addr(): mess.addr()}
	mess.neighbors = make([]string, 0)
	mess.neighborSet = make(map[string]struct{})
	mess.seqs = make(map[string]uint)
	mess.rumors = make(map[string][]types.Rumor)
	mess.Logger = _logger.With().Str("Messager", mess.sock.GetAddress()).Logger()
	mess.ackFutures = make(map[string]chan int)
	return mess
}

func (n *Messager) addr() string {
	return n.sock.GetAddress()
}

// Note: msg_ has to be a pointer
func (n *Messager) unicastTypesMsg(dest string, msg_ types.Message) error {
	msg, err := n.msgRegistry.MarshalMessage(msg_)
	if err != nil {
		return err
	}
	return n.Unicast(dest, msg)
}

// Unicast implements peer.Messaging
// send to itself is naturally supported by UDP
func (n *Messager) Unicast(dest string, msg transport.Message) error {
	// assemble a packet
	// relay shall be self
	relay := n.addr()
	header := transport.NewHeader(n.addr(), relay, dest, 0)
	pkt := transport.Packet{Header: &header, Msg: &msg}

	nextDest, err := n.send(pkt)
	if err != nil {
		err = fmt.Errorf("Unicast error: %w", err)
		n.Err(err).Send()
	}
	n.Debug().Str("dest", dest).Str("nextDest", nextDest).Str("msg", msg.String()).Str("pkt", pkt.String()).Msg("unicast packet sended")
	return err
}

// Broadcast sends a packet to all know destinations
// must not send the message to itself
// but still process it
func (n *Messager) Broadcast(msg transport.Message) error {
	n.Debug().Msg("start to broadcast")

	// 1. wrap a RumorMessage, and send it through the socket to one random neighbor
	// once the seq is added and Rumor is constructed, this Rumor is gurantted to
	// be sent(whether by broadcast or statusMsg)

	n.seqMu.Lock()
	// fetch my last seq and increase it
	if _, ok := n.seqs[n.addr()]; !ok {
		n.seqs[n.addr()] = 0
		n.rumors[n.addr()] = []types.Rumor{}
	}
	seq := n.seqs[n.addr()] + 1
	n.seqs[n.addr()] = seq

	// update rumors
	ru := types.Rumor{Origin: n.addr(), Msg: &msg, Sequence: uint(seq)}
	n.rumors[n.addr()] = append(n.rumors[n.addr()], ru)
	n.Debug().Str("seqs", fmt.Sprintf("%v", n.seqs)).Str("rumors", fmt.Sprintf("%v", n.rumors)).Msg("update seqs and rumors first")
	n.seqMu.Unlock()

	ruMsg, err := n.msgRegistry.MarshalMessage(&types.RumorsMessage{Rumors: []types.Rumor{ru}})
	if err != nil {
		n.Err(err).Send()
		return fmt.Errorf("Broadcast error: %w", err)
	}

	// send and wait for the ack
	go func() {
		if !n.hasNeighbor() {
			n.Warn().Msg("no neighbor, cannot broadcast, direct return")
		}
		preNei := ""
		tried := 0
		acked := false
		processed := false
		for tried < 2 && !acked {
			tried++
			// ensure the randNeigh is not previous one
			// if has only one neighbor, then randNeighExcept will return this only neighbor
			randNei := n.randNeighExcept(preNei)
			header := transport.NewHeader(n.addr(), n.addr(), randNei, 0)
			pkt := transport.Packet{Header: &header, Msg: &ruMsg}
			preNei = randNei
			// create and register the future before send, such that AckCallback will always happens after future register
			// create ack future, it is a buffered channel, such that ack after timeout do not block on sending on future
			future := make(chan int, 1)
			n.acuMu.Lock()
			n.ackFutures[pkt.Header.PacketID] = future
			n.Debug().Msgf("broadcast register a future for packet %s", pkt.Header.PacketID)
			n.acuMu.Unlock()

			n.Debug().Msgf("broadcast prepares to send pkt to %s", randNei)
			nextDest, err := n.send(pkt)
			if !processed {
				processed = true
				// 0. process the embeded message
				_header := transport.NewHeader(n.addr(), n.addr(), n.addr(), 0)
				err = n.msgRegistry.ProcessPacket(transport.Packet{
					Header: &_header,
					Msg:    &msg,
				})
				if err != nil {
					n.Err(fmt.Errorf("Broadcast process local error: %w", err)).Send()
					// return fmt.Errorf("Broadcast error: %w", err)
				}
				n.Debug().Msg("process Broad Msg done")
			}
			if err != nil {
				n.Err(fmt.Errorf("Broadcast error: %w", err)).Send()
				// FIXME: this early return did not delete entries
				return
			}
			n.Debug().Str("dest", header.Destination).Str("nextDest", nextDest).Str("msg", ruMsg.String()).Str("pkt", pkt.String()).Msg("possibly sended")

			// start to wait for the ack message
			n.Debug().Msgf("start to wait for broadcast ack message on packet %s", pkt.Header.PacketID)
			timeout := n.conf.AckTimeout
			if timeout == 0 {
				timeout = time.Duration(math.MaxInt32 * time.Millisecond)
			}
			timer := time.After(timeout)
			select {
			case <-future:
				acked = true
				n.Debug().Msgf("ack received and properly processed")
			case <-timer:
				n.Debug().Msgf("ack timeout, start another probe")
				// send to another random neighbor
			}
			// delete unused future
			n.acuMu.Lock()
			delete(n.ackFutures, pkt.Header.PacketID)
			n.acuMu.Unlock()
		}

	}()

	return nil
}

// Unicast implements peer.Messaging
// send to itself is naturally supported by UDP
func (n *Messager) strictUnicast(dest string, msg transport.Message) error {
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
func (n *Messager) strictSend(pkt transport.Packet) (string, error) {
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
func (n *Messager) strictNextHop(dest string) (string, error) {
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
func (n *Messager) forceSend(peer string, pkt transport.Packet) error {
	// send the pkt
	err := n.sock.Send(peer, pkt, 0)
	if err != nil {
		return err
	}
	return nil
}

// blocking send a packet, target is decided by the routing table
// return `nextDest` and error
func (n *Messager) send(pkt transport.Packet) (string, error) {
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

func (n *Messager) nextHop(dest string) (string, error) {
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
func (n *Messager) AddPeer(addr ...string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	// we could directly reach the peers
	// NOTE: adding ourselves should have no effects
	n.Info().Strs("peers", addr).Msg("adding peers")
	for i := 0; i < len(addr); i++ {
		n.route[addr[i]] = addr[i]
		if _, ok := n.neighborSet[addr[i]]; !ok && addr[i] != n.addr() {
			n.neighborSet[addr[i]] = struct{}{}
			n.neighbors = append(n.neighbors, addr[i])
		}

	}
	n.Trace().Str("route", n.route.String()).Str("neighbors", fmt.Sprintf("%v", n.neighbors)).Msg("after added")
}

func (n *Messager) addNeighbor(addr ...string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	// we could directly reach the peers
	// NOTE: adding ourselves should have no effects
	n.Trace().Strs("peers", addr).Msg("adding neis")
	for i := 0; i < len(addr); i++ {
		if _, ok := n.neighborSet[addr[i]]; !ok && addr[i] != n.addr() {
			n.neighborSet[addr[i]] = struct{}{}
			n.neighbors = append(n.neighbors, addr[i])
		}

	}
	n.Trace().Str("neighbors", fmt.Sprintf("%v", n.neighbors)).Msg("after neighbor added")
}

// GetRoutingTable implements peer.Service
func (n *Messager) GetRoutingTable() peer.RoutingTable {
	n.mu.Lock()
	defer n.mu.Unlock()
	copy := make(map[string]string)
	for k, v := range n.route {
		copy[k] = v
	}
	return copy
}

// SetRoutingEntry implements peer.Service
func (n *Messager) SetRoutingEntry(origin, relayAddr string) {
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
func (n *Messager) randNeigh() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.neighbors) == 0 {
		return NONEIGHBOR
	}
	return n.neighbors[rand.Int31n(int32(len(n.neighbors)))]
}

// Q: what if we only have one random neighbor? A: just return it
func (n *Messager) randNeighExcept(except string) string {
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

func (n *Messager) hasNeighbor() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.neighbors) != 0
}

func (n *Messager) isNeighbor(addr string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, nei := range n.neighbors {
		if nei == addr {
			return true
		}
	}
	return false
}

func (n *Messager) getNeisExcept(excepts ...string) []string {
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

func (n *Messager) getNeis() []string {
	return n.getNeisExcept(NONEIGHBOR)
}
