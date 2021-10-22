package impl

import (
	"errors"
	"fmt"
	"math/rand"
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
var NONEIGHBOR string = "NONEIGHBOR"

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
	node := &node{sock: conf.Socket, msgRegistry: conf.MessageRegistry, id: uniqueID(), conf: conf}
	// init the routing table, add this.addr
	node.route = peer.RoutingTable{node.addr(): node.addr()}
	node.seqs = make(map[string]uint)
	node.rumors = make(map[string][]types.Rumor)
	node.ackFutures = make(map[string]chan int)
	node.neighbors = make([]string, 0)
	node.neighborSet = make(map[string]struct{})

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
	conf        peer.Configuration

	// id          xid.ID
	// unique id, xid.ID is not human friendly
	id int32

	acuMu      sync.Mutex
	ackFutures map[string]chan int

	mu          sync.Mutex // protect access to `route` and `neighbors`, for example, listenDaemon and Unicast, redirect will access it
	neighbors   []string   // it will only grow, since no node will leave the network in assumption
	neighborSet map[string]struct{}
	route       peer.RoutingTable

	stat int32

	seqMu  sync.Mutex               // protect seqs and rumors
	seqs   map[string]uint          // rumor seq of other nodes, here nodes are not necessarily the neighbors, since rumor corresponds to one origin
	rumors map[string][]types.Rumor // key: node_addr value: rumors in increasing order of seq
}

func (n *node) statusReportDaemon(interval time.Duration) {
	if interval == 0 {
		n.Warn().Msg("status report is not activated since input interval is 0")
		return
	}

	for !n.isKilled() {
		time.Sleep(interval)

		if !n.hasNeighbor() {
			n.Warn().Msg("no neighbor, cannot send statusMsg periodically")
			continue
		}

		// MarshalMessage would access to the n.seqs, lock to protect
		n.seqMu.Lock()
		status := types.StatusMessage(n.seqs)
		statusMsg, err := n.msgRegistry.MarshalMessage(&status)
		n.seqMu.Unlock()

		if err != nil {
			n.Err(err).Msg("status report failed")
			continue
		}

		randNei := n.randNeigh()
		n.Info().Msgf("status report once to %s", randNei)
		if err = n.Unicast(randNei, statusMsg); err != nil {
			n.Err(err).Msg("status report failed")
			continue
		}
	}
}

func (n *node) heartbeatDaemon(interval time.Duration) {
	if interval == 0 {
		n.Warn().Msg("heartbeat is not activated since input interval is 0")
		return
	}

	for !n.isKilled() {
		n.Info().Msg("heartbeat once")
		empty, _ := n.msgRegistry.MarshalMessage(&types.EmptyMessage{})
		if err := n.Broadcast(empty); err != nil {
			n.Err(err).Send()
		}
		time.Sleep(interval)
	}
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
			n.Err(err).Msg("error while listening to socket")
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
		n.addNeighbor(pack.Header.RelayedBy) // we can only ensure that relay is near us

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
			pack.Header.RelayedBy = n.addr() // FIXME: not good, packet shall be immutable
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
	n.msgRegistry.RegisterMessageCallback(types.RumorsMessage{}, n.RumorsMsgCallback)
	n.Info().Msg("register callback for `RumorsMessage`")
	n.msgRegistry.RegisterMessageCallback(types.StatusMessage{}, n.StatusMsgCallback)
	n.Info().Msg("register callback for `StatusMessage`")
	n.msgRegistry.RegisterMessageCallback(types.AckMessage{}, n.AckMsgCallback)
	n.Info().Msg("register callback for `AckMessage`")
	n.msgRegistry.RegisterMessageCallback(types.EmptyMessage{}, types.EmptyMsgCallback)
	n.Info().Msg("register callback for `EmptyMessage`")
	n.msgRegistry.RegisterMessageCallback(types.PrivateMessage{}, n.PrivateMsgCallback)
	n.Info().Msg("register callback for `PrivateMessage`")
	// start a listining daemon to listen on the incoming message with `sock`
	n.Info().Msg("loading daemons...")
	go n.listenDaemon()
	go n.statusReportDaemon(n.conf.AntiEntropyInterval)
	go n.heartbeatDaemon(n.conf.HeartbeatInterval)
	n.Info().Msg("daemons loaded")
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

	// send the pkt
	err = n.sock.Send(nextDest, pkt, 0)
	if err != nil {
		return nextDest, fmt.Errorf("send error: %w", err)
	}
	return nextDest, nil
}

func (n *node) sendTimeout(pkt transport.Packet, timeout time.Duration) (string, error) {
	done := make(chan int, 1)
	var nextDest string
	var err error
	go func() {
		nextDest, err = n.send(pkt)
		done <- 0
	}()
	select {
	case <-done:
		if err != nil {
			return nextDest, fmt.Errorf("send error: %w", err)
		}
		return nextDest, nil
	case <-time.After(timeout):
		return "", transport.TimeoutErr(timeout)
	}
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
		n.Err(err).Send()
	}
	n.Debug().Str("dest", dest).Str("nextDest", nextDest).Str("msg", msg.String()).Str("pkt", pkt.String()).Msg("unicast packet sended")
	return err
}

// Broadcast sends a packet to all know destinations
// must not send the message to itself
// but still process it
func (n *node) Broadcast(msg transport.Message) error {
	// TODO: can I first process the message?
	n.Debug().Msg("start to broadcast")
	// 0. process the embeded message
	_header := transport.NewHeader(n.addr(), n.addr(), n.addr(), 0)
	err := n.msgRegistry.ProcessPacket(transport.Packet{
		Header: &_header,
		Msg:    &msg,
	})
	if err != nil {
		n.Err(err).Send()
		return fmt.Errorf("Broadcast error: %w", err)
	}
	n.Debug().Msg("process Broad Msg done")

	// 1. wrap a RumorMessage, and send it through the socket to one random neighbor
	// once the seq is added and Rumor is constructed, this Rumor is gurantted to
	// be sent out(ACK). So we dont need to worry about inconsistency of seq.
	// TODO: but there still some send/marshal error that would cause the problem of early return

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

	if !n.hasNeighbor() {
		n.Warn().Msg("no neighbor, cannot broadcast, direct return")
		return nil
	}

	// TODO: MarshalMessage 有问题是否应该直接 panic?
	ruMsg, err := n.msgRegistry.MarshalMessage(&types.RumorsMessage{Rumors: []types.Rumor{ru}})
	if err != nil {
		n.Err(err).Send()
		return fmt.Errorf("Broadcast error: %w", err)
	}
	preNei := ""
	acked := false
	for !acked {
		// TODO: what should be ttl?
		// ensure the randNeigh is not previous one
		// if has only one neighbor, then randNeighExcept will return this only neighbor
		randNei := n.randNeighExcept(preNei)
		header := transport.NewHeader(n.addr(), n.addr(), randNei, 0)
		pkt := transport.Packet{Header: &header, Msg: &ruMsg}
		preNei = randNei
		// create and register the future before send, such that AckCallback will always happens after future register
		// create ack future, it is a buffered channel, such that ack after timeout do not block on sending on future
		// TODO: 总结一下, 差点又犯了 happens before 的假设错误
		future := make(chan int, 1)
		n.acuMu.Lock()
		n.ackFutures[pkt.Header.PacketID] = future
		n.Debug().Msgf("broadcast register a future for packet %s", pkt.Header.PacketID)
		n.acuMu.Unlock()

		n.Debug().Msgf("broadcast prepares to send pkt to %s", randNei)
		nextDest, err := n.send(pkt)
		if err != nil {
			n.Err(err).Send()
			// TODO: this early return did not delete entries
			return fmt.Errorf("Broadcast error: %w", err)
		}
		n.Debug().Str("dest", header.Destination).Str("nextDest", nextDest).Str("msg", ruMsg.String()).Str("pkt", pkt.String()).Msg("possibly sended")

		// start to wait for the ack message
		n.Debug().Msgf("start to wait for broadcast ack message on packet %s", pkt.Header.PacketID)
		select {
		case <-future:
			n.Debug().Msgf("ack received and properly processed")
			acked = true
		case <-time.After(n.conf.AckTimeout):
			n.Debug().Msgf("ack timeout, start another probe")
			// send to another random neighbor
		}
		// delete unused future
		n.acuMu.Lock()
		delete(n.ackFutures, pkt.Header.PacketID)
		n.acuMu.Unlock()
	}
	return nil
}

// TODO: neighbor add should deduplicate
// TODO: status should only contains valid(cannot [])
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
	n.Debug().Str("route", n.route.String()).Str("neighbors", fmt.Sprintf("%v", n.neighbors)).Msg("after added")
}

func (n *node) addNeighbor(addr ...string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	// we could directly reach the peers
	// NOTE: adding ourselves should have no effects
	n.Info().Strs("peers", addr).Msg("adding peers")
	for i := 0; i < len(addr); i++ {
		if _, ok := n.neighborSet[addr[i]]; !ok {
			n.neighborSet[addr[i]] = struct{}{}
			n.neighbors = append(n.neighbors, addr[i])
		}

	}
	n.Debug().Str("neighbors", fmt.Sprintf("%v", n.neighbors)).Msg("after neighbor added")
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

// assumption: only ListenDaemon could exec RumorsMsgCallback;
// Q: possible race condition on packet?
// A: no, only possible inplace modificiation of packet happens after RumorMsgCallback processed.
// seqs 讨论: 可能的 race 来自于 BroadCast 对 self.seqID 的更新
// rumor process 讨论: 每一个 origin 的 rumor 一定是按序 process 的, 并且不会有任何的 race
// rumor: it only embed higher-level message. It will not embed rumors, status, ack etc.

// 这个方法非常关键, 因为涉及到 seqs 相关的更新, 处理不好就会造成 inconsistency
// Q: seqs update shall onReceived or onProcessed?
// A: currently onReceived. Because received req is guaranteed to be processed eventually.
func (n *node) RumorsMsgCallback(msg types.Message, pkt transport.Packet) error {
	__logger := n.Logger.With().Str("func", "RumorsMsgCallback").Logger()
	__logger.Info().Msg("enter rumors callback")

	// 1. processing each rumor
	rumorsMsg := msg.(*types.RumorsMessage)
	isNew := false
	for _, rumor := range rumorsMsg.Rumors {
		expected := false

		// here we access and possibly modify seqs, so we need to lock
		n.seqMu.Lock()
		if lastSeq, ok := n.seqs[rumor.Origin]; ok {
			if lastSeq+1 == rumor.Sequence {
				expected = true
				n.seqs[rumor.Origin] = rumor.Sequence // update lastSeq of this origin
				n.rumors[rumor.Origin] = append(n.rumors[rumor.Origin], rumor)
				__logger.Info().Str("Origin", rumor.Origin).Uint("Seq", rumor.Sequence).
					Msgf("expected seq")
			} else {
				__logger.Info().Str("Origin", rumor.Origin).Uint("Seq", rumor.Sequence).
					Msgf("UNexpected seq!!")
			}
		} else {
			// this is the first time we see this origin, so we need to restrict it to 1
			if rumor.Sequence == 1 {
				expected = true
				n.seqs[rumor.Origin] = 1                      // update lastSeq of this origin
				n.rumors[rumor.Origin] = []types.Rumor{rumor} // init + append
				__logger.Info().Str("Origin", rumor.Origin).Uint("Seq", rumor.Sequence).
					Msgf("expected seq")
			} else {
				__logger.Info().Str("Origin", rumor.Origin).Uint("Seq", rumor.Sequence).
					Msgf("UNexpected seq!!")
			}
		}
		n.seqMu.Unlock()

		// just ignore the message
		if !expected {
			continue
		}

		// the rumor is expected and valid
		isNew = true // there is some new msgs we need to process, so it is new

		// update the routing table
		// if !n.isNeighbor(rumor.Origin) {
		// 	__logger.Info().Msg("%s is not neighbor, we could update routing table")
		// }
		n.SetRoutingEntry(rumor.Origin, pkt.Header.RelayedBy)

		// now process the embed msg, call the callback
		// wrap a packet
		pkt := transport.Packet{
			Header: pkt.Header,
			Msg:    rumor.Msg,
		}
		__logger.Info().Msg("now processing embed message")
		if err := n.msgRegistry.ProcessPacket(pkt); err != nil {
			__logger.Err(err).Send()
			return fmt.Errorf("RumorsMsgCallback fail: processing rumor's embed msg fail: %w", err)
		}
	}

	// Q: why after processing rumors?
	// A: because rumors might contain the updates sender wants to give us. If we directly ack
	//    then sender will still find we have not the update, then it will send it again.
	// 2. send back a AckMessage to source
	// here we need to lock since the MarshalMessage call will read n.seqs
	n.seqMu.Lock()
	statusMsg := types.StatusMessage(n.seqs)
	ack, err := n.msgRegistry.MarshalMessage(&types.AckMessage{AckedPacketID: pkt.Header.PacketID, Status: statusMsg})
	n.seqMu.Unlock()

	if err != nil {
		__logger.Err(err).Send()
		return fmt.Errorf("RumorsMsgCallback fail: %w", err)
	}
	__logger.Info().Str("dest", pkt.Header.RelayedBy).Msg("send ack back")
	// Q: shall it be creator or relayBy?
	// A: it should be relayedBy; In broadcast init case, both relayedBy and source is ok.
	//    in broadcast propagation case, the sender will not check ack. We use relayedBy
	//    since it is in the routing table while source might not. In other words, Ack is point2point.
	err = n.Unicast(pkt.Header.RelayedBy, ack)
	if err != nil {
		__logger.Err(err).Send()
		return fmt.Errorf("RumorsMsgCallback fail: %w", err)
	}

	// 3. possibly redirect the RumorsMessage to another node incase it is "new"
	// for those msgs that are ignored, are they not new. see https://moodle.epfl.ch/mod/forum/discuss.php?d=65056
	if !isNew {
		__logger.Info().Msg("nothing new from rumors, dont need to propagate, return")
		return nil
	}

	// TODO: what about the ttl?
	// here we create a new packet and use this node as source, since this is a re-send rather than routing
	// dont need to check neighbor, otherwise we cannot receive this message.
	randNei := n.randNeighExcept(pkt.Header.RelayedBy)
	if randNei == pkt.Header.RelayedBy {
		__logger.Warn().Str("callback", "RumorsMsgCallback").Msg("has only one neighbor, skip Rumor propagation")
		return nil
	}
	__logger.Info().Msgf("something new, prepares to unicast the rumores to %s", randNei)
	err = n.Unicast(randNei, *pkt.Msg)
	if err != nil {
		__logger.Err(err).Send()
		return fmt.Errorf("RumorsMsgCallback fail: propagate to random neighbor fail: %w", err)
	}
	return nil
}

// assumption: only ListenDaemon invoke StatusMsgCallback. That is, it will not be called with Unicast/BroadCast, thus will never be embeded
// so it is generally race-free
func (n *node) StatusMsgCallback(msg types.Message, pkt transport.Packet) error {
	n.Debug().Msgf("enter status callback")
	other := pkt.Header.Source
	statusMsg := msg.(*types.StatusMessage)
	otherView := map[string]uint(*statusMsg)
	meExceptOther := map[string][]types.Rumor{}
	otherExceptMe := false

	// ensure a consistent view
	n.seqMu.Lock()
	myView := n.seqs
	// marshal earlier to keep a consistent view
	possibleStatus := types.StatusMessage(n.seqs)
	possibleStatusMsg, marshalErr := n.msgRegistry.MarshalMessage(&possibleStatus)
	// compare myView with otherView
	for p, lastSeq := range myView {
		if otherLastSeq, ok := otherView[p]; ok {
			// other view also contains peer, then compare the seq
			if lastSeq < otherLastSeq {
				// other have something we dont have
				n.Debug().Msgf("I dont have rumors(seq %d-%d) from peer %s", lastSeq+1, otherLastSeq+1, p)
				otherExceptMe = true

			} else if lastSeq > otherLastSeq {
				// we have something others dont have
				meExceptOther[p] = make([]types.Rumor, lastSeq-otherLastSeq)
				copy(meExceptOther[p], n.rumors[p][otherLastSeq:lastSeq])
				n.Debug().Msgf("peer %s does not have rumors(seq %d-%d) from peer %s", pkt.Header.Source, otherLastSeq+1, lastSeq+1, p)
			}
		} else {
			// we have something others dont have
			// others does not contain any Rumors of the peer p
			meExceptOther[p] = make([]types.Rumor, lastSeq)
			copy(meExceptOther[p], n.rumors[p])
			n.Debug().Msgf("peer %s does not have any rumors from peer %s", pkt.Header.Source, p)
		}
	}

	// compare otherView with myView
	for p := range otherView {
		if _, ok := myView[p]; !ok {
			// other have something we dont have
			n.Debug().Msgf("I dont have rumors(seq 1-%d) from peer %s", otherView[p], p)
			otherExceptMe = true
		}
		// when ok=true, it is common key(p), this case has already been handled, just skip
	}
	n.seqMu.Unlock()
	n.Debug().Msgf("meExceptOther: %v", meExceptOther)
	n.Debug().Msgf("otherExceptMe: %v", otherExceptMe)

	if otherExceptMe {
		// send StatusMessage to other, such that other would send my-missing rumors back to me
		if marshalErr != nil {
			n.Err(marshalErr).Send()
			return fmt.Errorf("StatusMsgCallback fail: send status message back: %w", marshalErr)
		}
		n.Info().Msgf("otherExceptMe valid, unicast statusMsg %s to %s", possibleStatus, other)
		if err := n.Unicast(other, possibleStatusMsg); err != nil {
			n.Err(err).Send()
			return fmt.Errorf("StatusMsgCallback fail: send status message back: %w", err)
		}
	}

	if len(meExceptOther) > 0 {
		// send RumorsMessage to other, which consists of missing rumors of others
		otherMissingRumors := make([]types.Rumor, 0, 10)
		for _, rus := range meExceptOther {
			otherMissingRumors = append(otherMissingRumors, rus...)
		}
		_rumorsMsg := types.RumorsMessage{Rumors: otherMissingRumors}
		rumorsMsg, err := n.msgRegistry.MarshalMessage(&_rumorsMsg)
		if err != nil {
			n.Err(err).Send()
			return fmt.Errorf("StatusMsgCallback fail: send Rumors message: %w", err)
		}
		n.Info().Msgf("meExceptOther valid, unicast rumorsMsg %s to %s", _rumorsMsg, other)
		if err := n.Unicast(other, rumorsMsg); err != nil {
			n.Err(err).Send()
			return fmt.Errorf("StatusMsgCallback fail: send Rumors message: %w", err)
		}

	}

	// me and other has exactly the same view
	if len(meExceptOther) == 0 && !otherExceptMe && rand.Float64() < n.conf.ContinueMongering {
		// "ContinueMongering"
		// send to a random nei other than other
		// Note: dont need to check neight existence, since we receive a status from a connected node
		nei := n.randNeighExcept(other)
		if nei == other {
			n.Warn().Str("callback", "statusMsg").Msg("only one neighbor, dont need to propagate the consistent status/view")
			return nil
		}
		if marshalErr != nil {
			n.Err(marshalErr).Send()
			return fmt.Errorf("StatusMsgCallback fail: send status message rand: %w", marshalErr)
		}
		n.Info().Msgf("same view, unicast statusMsg %s to random nei %s", statusMsg, other)
		if err := n.Unicast(nei, possibleStatusMsg); err != nil {
			n.Err(err).Send()
			return fmt.Errorf("StatusMsgCallback fail: send status message rand: %w", err)
		}
	}

	return nil
}

// Q: who shall do the gc on ackFutures?
// 		1. -> Broadcast: at timeout, we delete this callback
//         pro: add and delete in one routine
//		   con: ack might arrive later, it will not find the future
//      2. ~~AckMsgCallback~~: after it wake up future
//         pro: cleaner for Ack
//         con: if a message is lost in the network, then no ack will be received, and we got a ghost entry
// assumption: only ListenDaemon invoke AckMsgCallback. That is, it will not be called with Unicast/BroadCast, thus will never be embeded
func (n *node) AckMsgCallback(msg types.Message, pkt transport.Packet) error {
	n.Debug().Msg("start ack callback")
	ack := msg.(*types.AckMessage)
	n.acuMu.Lock()
	future, ok := n.ackFutures[ack.AckedPacketID]
	n.acuMu.Unlock()

	if ok {
		future <- 0
	} else {
		// do nothing, it is a arrive-late ack msg
		n.Info().Msgf("packet %s has no future to complete", pkt.Header.PacketID)
	}

	// TODO: if already timeouted, shall we process the StatusMessage?
	// TODO: too many Marshalcall and error checking
	status, err := n.msgRegistry.MarshalMessage(&ack.Status)
	if err != nil {
		n.Err(err).Send()
		return fmt.Errorf("AckMsgCallback fail: %w", err)
	}
	err = n.msgRegistry.ProcessPacket(
		transport.Packet{
			Header: pkt.Header,
			Msg:    &status,
		},
	)
	if err != nil {
		n.Err(err).Send()
		return fmt.Errorf("AckMsgCallback fail: %w", err)
	}
	n.Debug().Msg("ack process embeded msg done")

	return nil
}

// TODO: callbacks shall not print, rather just return the err
func (n *node) PrivateMsgCallback(msg types.Message, pkt transport.Packet) error {
	private := msg.(*types.PrivateMessage)
	if _, ok := private.Recipients[n.addr()]; !ok {
		return nil
	}

	newPkt := transport.Packet{
		Header: pkt.Header,
		Msg:    private.Msg,
	}
	if err := n.msgRegistry.ProcessPacket(newPkt); err != nil {
		return fmt.Errorf("PrivateMsgCallback fail: %w", err)
	}
	return nil
}

// neighbors access shall be protected,
// TODO: shall it share the same lock with routeMutex?
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
