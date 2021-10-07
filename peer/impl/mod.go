package impl

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/registry"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

var _logger zerolog.Logger = zerolog.New(
	zerolog.NewConsoleWriter(
		func(w *zerolog.ConsoleWriter) { w.Out = os.Stderr },
		func(w *zerolog.ConsoleWriter) { w.TimeFormat = "15:04:05.000" })).Level(zerolog.DebugLevel).
	With().Timestamp().Logger()
var _peerCount int32 = -1

// peer state
const (
	KILL = iota
	ALIVE
)

func uniqueID() int32 {
	return atomic.AddInt32(&_peerCount, 1)
}

// NewPeer creates a new peer. You can change the content and location of this
// function but you MUST NOT change its signature and package location.
func NewPeer(conf peer.Configuration) peer.Peer {
	// time.Now().Format()
	// here you must return a struct that implements the peer.Peer functions.
	// Therefore, you are free to rename and change it as you want.
	node := &node{sock: conf.Socket, msgRegistry: conf.MessageRegistry, id: uniqueID()}
	// init the routing table, add this.addr
	node.route = peer.RoutingTable{node.addr(): node.addr()}
	node.Logger = _logger.With().Str("Peer", fmt.Sprintf("%d %s", node.id, node.sock.GetAddress())).Logger()
	_logger.Info().Int32("id", node.id).Str("addr", node.sock.GetAddress()).Msg("create new peer")
	return node
}

// node implements a peer to build a Peerster system
//
// - implements peer.Peer
type node struct {
	peer.Peer
	zerolog.Logger

	// You probably want to keep the peer.Configuration on this struct:
	sock        transport.Socket
	msgRegistry registry.Registry

	// id          xid.ID
	// unique id, xid.ID is not human friendly
	id int32

	mu    sync.Mutex // protect access to `route`, for example, listenDaemon and Unicast, redirect will access it
	route peer.RoutingTable

	stat int32
}

func (n *node) listenDaemon() {
	// 1. must check if the message is truly for the node
	// 	1.1 if yes, use `msgRegistry` to execute the callback associated with the message
	//  1.2 if no, update the `RelayedBy` field of the message
	timeout := 500 * time.Millisecond
	// while not killed
	for !n.isKilled() {
		pack, err := n.sock.Recv(timeout)
		var timeErr transport.TimeoutErr
		// timeout error
		if err != nil && errors.As(err, &timeErr) {
			// continue to receive
			n.Info().Msg("timeout, continue listening")
			continue
		}
		// other types of error
		if err != nil {
			n.Error().Err(err).Msg("error while listening to socket")
			continue
		}
		// start processing the pack
		n.Info().Str("pkt", pack.String()).Msg("receive packet")
		// 0. update the routing table
		//	0.1 now we might know some new peers: source(creator), relayedBy(sender) and destinition of the packet
		//	0.2 FIXME: we dont know any other infos for `SetRoutingEntry`? we only know that RelayedBy or Source -> me routing
		// FIXME: AddPeer only add neighbor nodes or not?
		// what do we know?
		// 	- relayedBy is near us relayedBy:relayedBy (if relayedBy is empty, then source is near us)
		n.AddPeer(pack.Header.RelayedBy) // we can only ensure that relay is near us

		// 1. must check if the message is truly for the node
		// 	1.1 if yes, use `msgRegistry` to execute the callback associated with the message
		//  1.2 if no, update the `RelayedBy` field of the message
		if pack.Header.Destination == n.sock.GetAddress() {
			n.Info().Str("addr", pack.Header.Destination).Msg("addr matched between peer and sender")
			if err := n.msgRegistry.ProcessPacket(pack); err != nil {
				n.Error().Err(err).Msg("error while processing the packet")
			}

		} else {
			n.Warn().Str("pack.addr", pack.Header.Destination).Str("this.addr", n.sock.GetAddress()).Msg("unmatched addr, set relay to my.addr and redirect the pkt")
			pack.Header.RelayedBy = n.addr()
			nextDest, err := n.send(pack)
			n.Err(err).Str("dest", pack.Header.Destination).Str("nextDest", nextDest).Str("msg", pack.Msg.String()).Str("pkt", pack.String()).Msg("relay packet")
		}
	}
}

// Start implements peer.Service
func (n *node) Start() error {
	n.Info().Msg("Starting...")
	n.stat = ALIVE
	n.msgRegistry.RegisterMessageCallback(types.ChatMessage{}, types.ChatMsgCallback)
	n.Info().Msg("register callback for `ChatMessage`")
	// start a listining daemon to listen on the incoming message with `sock`
	n.Info().Msg("loading listen daemon...")
	go n.listenDaemon()
	n.Info().Msg("listen daemon loaded")
	n.Info().Msg("Start done")
	return nil
}

// Stop implements peer.Service
func (n *node) Stop() error {
	// panic("to be implemented in HW0")
	n.Info().Msg("Stoping...")
	atomic.StoreInt32(&n.stat, KILL)
	// FIXME: shall we close the socket?
	return nil
}

func (n *node) nextHop(dest string) (string, error) {
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

// send a packet, target is decided by the routing table
// return `nextDest` and error
func (n *node) send(pkt transport.Packet) (string, error) {
	// 1. source should not be changed
	// 2. relay=me
	// 3. dest should de decided by the routing table
	nextDest, err := n.nextHop(pkt.Header.Destination)
	if err != nil {
		return nextDest, fmt.Errorf("send error: %w", err)
	}

	// send the pkt
	timeout := 1 * time.Second // FIXME: shall we add this timeout?
	err = n.sock.Send(nextDest, pkt, timeout)
	if err != nil {
		return nextDest, fmt.Errorf("send error: %w", err)
	}
	return nextDest, nil
}

// Unicast implements peer.Messaging
// send to itself is naturally supported by UDP
func (n *node) Unicast(dest string, msg transport.Message) error {
	// assemble a packet
	// relay shall be self
	relay := n.addr()
	header := transport.NewHeader(n.addr(), relay, dest, 0)
	pkt := transport.Packet{Header: &header, Msg: &msg}

	nextDest, err := n.send(pkt)
	if err != nil {
		err = fmt.Errorf("Unicast error: %w", err)
		n.Err(err)
	}
	n.Info().Str("dest", dest).Str("nextDest", nextDest).Str("msg", msg.String()).Str("pkt", pkt.String()).Msg("send packet success")
	return err
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
	}
	n.Debug().Str("route", n.route.String()).Msg("after added")
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

func (n *node) isKilled() bool {
	return atomic.LoadInt32(&n.stat) == KILL
}

func (n *node) addr() string {
	return n.sock.GetAddress()
}
