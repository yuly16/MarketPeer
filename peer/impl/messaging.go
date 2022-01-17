package impl

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/registry"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

// Messager implements peer.Messager
type Messager struct {
	peer.Messager

	sock transport.Socket
	zerolog.Logger
	msgRegistry registry.Registry
	conf        peer.Configuration

	stat int32

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

func (m *Messager) heartbeatDaemon(interval time.Duration) {
	if interval == 0 {
		m.Warn().Msg("heartbeat is not activated since input interval is 0")
		return
	}

	for !m.isKilled() {
		m.Info().Msg("heartbeat once")
		empty, _ := m.msgRegistry.MarshalMessage(&types.EmptyMessage{})
		if err := m.Broadcast(empty); err != nil {
			m.Err(err).Send()
		}
		time.Sleep(interval)
	}
}

func (m *Messager) statusReportDaemon(interval time.Duration) {
	if interval == 0 {
		m.Warn().Msg("status report is not activated since input interval is 0")
		return
	}

	for !m.isKilled() {
		time.Sleep(interval)

		if !m.hasNeighbor() {
			m.Warn().Msg("no neighbor, cannot send statusMsg periodically")
			continue
		}

		// MarshalMessage would access to the n.seqs, lock to protect
		m.seqMu.Lock()
		status := types.StatusMessage(m.seqs)
		statusMsg, err := m.msgRegistry.MarshalMessage(&status)
		m.seqMu.Unlock()

		if err != nil {
			m.Err(err).Msg("status report failed")
			continue
		}

		randNei := m.randNeigh()
		m.Trace().Msgf("status report once to %s", randNei)
		if err = m.Unicast(randNei, statusMsg); err != nil {
			m.Err(err).Msg("status report failed")
			continue
		}
	}
}

// TODO: make it the method of messager
func (m *Messager) listenDaemon() {
	// 1. must check if the message is truly for the node
	// 	1.1 if yes, use `msgRegistry` to execute the callback associated with the message
	//  1.2 if no, update the `RelayedBy` field of the message
	timeout := 500 * time.Millisecond
	// while not killed
	for !m.isKilled() {
		pack, err := m.sock.Recv(timeout)
		var timeErr transport.TimeoutErr
		// timeout error
		if err != nil && errors.As(err, &timeErr) {
			// continue to receive
			m.Trace().Msg("timeout, continue listening")
			continue
		}
		// other types of error
		if err != nil {
			m.Err(err).Msg("error while listening to socket")
			continue
		}
		// start processing the pack
		m.Debug().Str("pkt", pack.String()).Msg("receive packet")
		// 0. update the routing table
		m.addNeighbor(pack.Header.RelayedBy) // we can only ensure that relay is near us

		// 1. must check if the message is truly for the node
		// 	1.1 if yes, use `msgRegistry` to execute the callback associated with the message
		//  1.2 if no, update the `RelayedBy` field of the message
		if pack.Header.Destination == m.sock.GetAddress() {
			// fmt.Printf("I receive a msg from %s\n", pack.Header.Source)
			m.Trace().Str("addr", pack.Header.Destination).Msg("addr matched between peer and sender")
			if err := m.msgRegistry.ProcessPacket(pack); err != nil {
				var sendErr *SenderCallbackError
				if !errors.As(err, &sendErr) { // we only care about non-sender error
					m.Error().Err(err).Msg("error while processing the packet")
				}
			}

		} else {
			m.Warn().Str("pack.addr", pack.Header.Destination).Str("this.addr", m.sock.GetAddress()).Msg("unmatched addr, set relay to my.addr and redirect the pkt")
			pack.Header.RelayedBy = m.addr() // FIXME: not good, packet shall be immutable
			nextDest, err := m.send(pack)
			m.Err(err).Str("dest", pack.Header.Destination).Str("nextDest", nextDest).Str("msg", pack.Msg.String()).Str("pkt", pack.String()).Msg("relay packet")
		}
	}
}

func (m *Messager) Start() error {
	m.stat = ALIVE
	m.Info().Msg("loading daemons...")

	m.msgRegistry.RegisterMessageCallback(types.RumorsMessage{}, m.RumorsMsgCallback)
	m.Trace().Msg("register callback for `RumorsMessage`")
	m.msgRegistry.RegisterMessageCallback(types.StatusMessage{}, m.StatusMsgCallback)
	m.Trace().Msg("register callback for `StatusMessage`")
	m.msgRegistry.RegisterMessageCallback(types.AckMessage{}, m.AckMsgCallback)
	m.Trace().Msg("register callback for `AckMessage`")
	m.msgRegistry.RegisterMessageCallback(types.EmptyMessage{}, types.EmptyMsgCallback)
	m.Trace().Msg("register callback for `EmptyMessage`")
	m.msgRegistry.RegisterMessageCallback(types.PrivateMessage{}, m.PrivateMsgCallback)
	m.Trace().Msg("register callback for `PrivateMessage`")

	go m.listenDaemon()
	go m.statusReportDaemon(m.conf.AntiEntropyInterval)
	go m.heartbeatDaemon(m.conf.HeartbeatInterval)
	m.Info().Msg("daemons loaded")
	return nil
}

func (m *Messager) Stop() error {
	atomic.StoreInt32(&m.stat, KILL)
	return nil
}

func (m *Messager) isKilled() bool {
	return atomic.LoadInt32(&m.stat) == KILL
}

func (m *Messager) addr() string {
	return m.sock.GetAddress()
}

// Note: msg_ has to be a pointer
func (m *Messager) unicastTypesMsg(dest string, msg_ types.Message) error {
	msg, err := m.msgRegistry.MarshalMessage(msg_)
	if err != nil {
		return err
	}
	return m.Unicast(dest, msg)
}

// Unicast implements peer.Messaging
// send to itself is naturally supported by UDP
func (m *Messager) Unicast(dest string, msg transport.Message) error {
	// assemble a packet
	// relay shall be self
	relay := m.addr()
	header := transport.NewHeader(m.addr(), relay, dest, 0)
	pkt := transport.Packet{Header: &header, Msg: &msg}

	nextDest, err := m.send(pkt)
	if err != nil {
		err = fmt.Errorf("Unicast error: %w, msg=%s", err, msg.String())
		m.Err(err).Send()
	}
	m.Debug().Str("dest", dest).Str("nextDest", nextDest).Str("msg", msg.String()).Str("pkt", pkt.String()).Msg("unicast packet sended")
	return err
}

// Broadcast sends a packet to all know destinations
// must not send the message to itself
// but still process it
func (m *Messager) Broadcast(msg transport.Message) error {
	m.Debug().Msg("start to broadcast")

	// 1. wrap a RumorMessage, and send it through the socket to one random neighbor
	// once the seq is added and Rumor is constructed, this Rumor is gurantted to
	// be sent(whether by broadcast or statusMsg)

	m.seqMu.Lock()
	// fetch my last seq and increase it
	if _, ok := m.seqs[m.addr()]; !ok {
		m.seqs[m.addr()] = 0
		m.rumors[m.addr()] = []types.Rumor{}
	}
	seq := m.seqs[m.addr()] + 1
	m.seqs[m.addr()] = seq

	// update rumors
	ru := types.Rumor{Origin: m.addr(), Msg: &msg, Sequence: uint(seq)}
	m.rumors[m.addr()] = append(m.rumors[m.addr()], ru)
	m.Debug().Str("seqs", fmt.Sprintf("%v", m.seqs)).Str("rumors", fmt.Sprintf("%v", m.rumors)).Msg("update seqs and rumors first")
	m.seqMu.Unlock()

	ruMsg, err := m.msgRegistry.MarshalMessage(&types.RumorsMessage{Rumors: []types.Rumor{ru}})
	if err != nil {
		m.Err(err).Send()
		return fmt.Errorf("Broadcast error: %w", err)
	}

	// send and wait for the ack
	go func() {
		if !m.hasNeighbor() {
			m.Warn().Msg("no neighbor, cannot broadcast, direct return")
			// 0. if no neighbour, then directly process the embeded message
			_header := transport.NewHeader(m.addr(), m.addr(), m.addr(), 0)

			err = m.msgRegistry.ProcessPacket(transport.Packet{
				Header: &_header,
				Msg:    &msg,
			})
			if err != nil {
				m.Err(err).Send()
			}
			return
		}
		preNei := ""
		tried := 0
		acked := false
		processed := false
		for tried < 2 && !acked {
			tried++
			// ensure the randNeigh is not previous one
			// if has only one neighbor, then randNeighExcept will return this only neighbor
			randNei := m.randNeighExcept(preNei)
			header := transport.NewHeader(m.addr(), m.addr(), randNei, 0)
			pkt := transport.Packet{Header: &header, Msg: &ruMsg}
			preNei = randNei
			// create and register the future before send, such that AckCallback will always happens after future register
			// create ack future, it is a buffered channel, such that ack after timeout do not block on sending on future
			future := make(chan int, 1)
			m.acuMu.Lock()
			m.ackFutures[pkt.Header.PacketID] = future
			m.Debug().Msgf("broadcast register a future for packet %s", pkt.Header.PacketID)
			m.acuMu.Unlock()

			m.Debug().Msgf("broadcast prepares to send pkt to %s", randNei)
			nextDest, err := m.send(pkt)
			if !processed {
				processed = true
				// 0. process the embeded message
				_header := transport.NewHeader(m.addr(), m.addr(), m.addr(), 0)
				err = m.msgRegistry.ProcessPacket(transport.Packet{
					Header: &_header,
					Msg:    &msg,
				})
				if err != nil {
					m.Err(fmt.Errorf("Broadcast process local error: %w", err)).Send()
					// return fmt.Errorf("Broadcast error: %w", err)
				}
				m.Debug().Msg("process Broad Msg done")
			}
			if err != nil {
				m.Err(fmt.Errorf("Broadcast error: %w", err)).Send()
				// FIXME: this early return did not delete entries
				return
			}
			m.Debug().Str("dest", header.Destination).Str("nextDest", nextDest).Str("msg", ruMsg.String()).Str("pkt", pkt.String()).Msg("possibly sended")

			// start to wait for the ack message
			m.Debug().Msgf("start to wait for broadcast ack message on packet %s", pkt.Header.PacketID)
			timeout := m.conf.AckTimeout
			if timeout == 0 {
				timeout = time.Duration(math.MaxInt32 * time.Millisecond)
			}
			timer := time.After(timeout)
			select {
			case <-future:
				acked = true
				m.Debug().Msgf("ack received and properly processed")
			case <-timer:
				m.Debug().Msgf("ack timeout, start another probe")
				// send to another random neighbor
			}
			// delete unused future
			m.acuMu.Lock()
			delete(m.ackFutures, pkt.Header.PacketID)
			m.acuMu.Unlock()
		}

	}()

	return nil
}

// Unicast implements peer.Messaging
// send to itself is naturally supported by UDP
func (m *Messager) strictUnicast(dest string, msg transport.Message) error {
	// assemble a packet
	// relay shall be self
	relay := m.addr()
	header := transport.NewHeader(m.addr(), relay, dest, 0)
	pkt := transport.Packet{Header: &header, Msg: &msg}

	nextDest, err := m.strictSend(pkt)
	if err != nil {
		err = fmt.Errorf("strictUnicast error: %w", err)
		m.Err(err).Send()
	}
	m.Debug().Str("dest", dest).Str("nextDest", nextDest).Str("msg", msg.String()).Str("pkt", pkt.String()).Msg("unicast packet sended")
	return err
}

// blocking send a packet, target is decided by the routing table
// return `nextDest` and error
func (m *Messager) strictSend(pkt transport.Packet) (string, error) {
	// 1. source should not be changed
	// 2. relay=me
	// 3. dest should de decided by the routing table
	nextDest, err := m.strictNextHop(pkt.Header.Destination)
	if err != nil {
		return nextDest, fmt.Errorf("send error: %w", err)
	}

	// send the pkt
	err = m.sock.Send(nextDest, pkt, 0)
	if err != nil {
		return nextDest, fmt.Errorf("send error: %w", err)
	}
	return nextDest, nil
}

// strictNextHop will not care about neighbors
func (m *Messager) strictNextHop(dest string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// dest must be known
	nextDest, ok := m.route[dest]
	var err error
	if !ok {
		err = fmt.Errorf("dest=%s is unknown to me=%s", dest, m.addr())
	}
	return nextDest, err
}

// no need to check routing table
func (m *Messager) forceSend(peer string, pkt transport.Packet) error {
	// send the pkt
	err := m.sock.Send(peer, pkt, 0)
	if err != nil {
		return err
	}
	return nil
}

// blocking send a packet, target is decided by the routing table
// return `nextDest` and error
func (m *Messager) send(pkt transport.Packet) (string, error) {
	// 1. source should not be changed
	// 2. relay=me
	// 3. dest should de decided by the routing table
	nextDest, err := m.nextHop(pkt.Header.Destination)
	if err != nil {
		return nextDest, fmt.Errorf("send error: %w", err)
	}
	// send the packet
	err = m.forceSend(nextDest, pkt)
	if err != nil {
		return nextDest, fmt.Errorf("send error: %w", err)
	}
	return nextDest, nil
}

func (m *Messager) nextHop(dest string) (string, error) {
	if m.isNeighbor(dest) {
		return dest, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if dest == NONEIGHBOR {
		fmt.Println("sgfdg")
	}
	// dest must be known
	nextDest, ok := m.route[dest]
	var err error
	if !ok {
		err = fmt.Errorf("dest=%s is unknown to me=%s", dest, m.addr())
	}
	return nextDest, err
}

// AddPeer implements peer.Service
func (m *Messager) AddPeer(addr ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// we could directly reach the peers
	// NOTE: adding ourselves should have no effects
	m.Info().Strs("peers", addr).Msg("adding peers")
	for i := 0; i < len(addr); i++ {
		m.route[addr[i]] = addr[i]
		if _, ok := m.neighborSet[addr[i]]; !ok && addr[i] != m.addr() {
			m.neighborSet[addr[i]] = struct{}{}
			m.neighbors = append(m.neighbors, addr[i])
		}

	}
	m.Trace().Str("route", m.route.String()).Str("neighbors", fmt.Sprintf("%v", m.neighbors)).Msg("after added")
}

func (m *Messager) addNeighbor(addr ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// we could directly reach the peers
	// NOTE: adding ourselves should have no effects
	m.Trace().Strs("peers", addr).Msg("adding neis")
	for i := 0; i < len(addr); i++ {
		if _, ok := m.neighborSet[addr[i]]; !ok && addr[i] != m.addr() {
			m.neighborSet[addr[i]] = struct{}{}
			m.neighbors = append(m.neighbors, addr[i])
		}

	}
	m.Trace().Str("neighbors", fmt.Sprintf("%v", m.neighbors)).Msg("after neighbor added")
}

// GetRoutingTable implements peer.Service
func (m *Messager) GetRoutingTable() peer.RoutingTable {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := make(map[string]string)
	for k, v := range m.route {
		copy[k] = v
	}
	return copy
}

// SetRoutingEntry implements peer.Service
func (m *Messager) SetRoutingEntry(origin, relayAddr string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// If relayAddr is empty then the record must be deleted
	if relayAddr == "" {
		delete(m.route, origin)
	} else {
		// simply overwrite
		m.route[origin] = relayAddr
	}
	m.Debug().Str("origin", origin).Str("relay", relayAddr).Msg("set routing entry")
	m.Debug().Str("route", m.route.String()).Msg("routing table after set")
}

// neighbors access shall be protected,
// FIXME: shall it share the same lock with routeMutex?
func (m *Messager) randNeigh() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.neighbors) == 0 {
		return NONEIGHBOR
	}
	return m.neighbors[rand.Int31n(int32(len(m.neighbors)))]
}

// Q: what if we only have one random neighbor? A: just return it
func (m *Messager) randNeighExcept(except string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.neighbors) == 0 {
		return NONEIGHBOR
	}
	if len(m.neighbors) == 1 {
		return m.neighbors[0]
	}
	randN := m.neighbors[rand.Int31n(int32(len(m.neighbors)))]
	for randN == except {
		randN = m.neighbors[rand.Int31n(int32(len(m.neighbors)))]
	}
	return randN
}

func (m *Messager) hasNeighbor() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.neighbors) != 0
}

func (m *Messager) isNeighbor(addr string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, nei := range m.neighbors {
		if nei == addr {
			return true
		}
	}
	return false
}

func (m *Messager) getNeisExcept(excepts ...string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	ret := make([]string, 0, len(m.neighbors))
	for _, nei := range m.neighbors {
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

func (m *Messager) getNeis() []string {
	return m.getNeisExcept(NONEIGHBOR)
}

func (m *Messager) GetNeighbors() []string {
	return m.getNeis()
}
