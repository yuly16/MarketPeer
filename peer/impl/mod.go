package impl

import (
	"bytes"
	"crypto"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/registry"
	"go.dedis.ch/cs438/storage"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

var _logger zerolog.Logger = zerolog.New(
	zerolog.NewConsoleWriter(
		func(w *zerolog.ConsoleWriter) { w.Out = os.Stderr },
		func(w *zerolog.ConsoleWriter) { w.TimeFormat = "15:04:05.000" })).Level(zerolog.ErrorLevel).
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
	node.blob = conf.Storage.GetDataBlobStore()
	node.nameing = conf.Storage.GetNamingStore()
	// init the routing table, add this.addr
	node.route = peer.RoutingTable{node.addr(): node.addr()}
	node.seqs = make(map[string]uint)
	node.rumors = make(map[string][]types.Rumor)
	node.ackFutures = make(map[string]chan int)
	node.neighbors = make([]string, 0)
	node.neighborSet = make(map[string]struct{})

	if node.conf.AckTimeout == 0 {
		node.conf.AckTimeout = math.MaxInt64
	}

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

	// storage
	blob    storage.Store
	nameing storage.Store

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

	// send the pkt
	err = n.sock.Send(nextDest, pkt, 0)
	if err != nil {
		return nextDest, fmt.Errorf("send error: %w", err)
	}
	return nextDest, nil
}

// func hexHash(data []byte) (string, error) {
// 	h := crypto.SHA256.New()
// 	if _, err := h.Write(data); err != nil {
// 		return "", err
// 	}
// 	return hex.EncodeToString(h.Sum(nil)), nil
// }

func sha256(data []byte) ([]byte, error) {
	h := crypto.SHA256.New()
	if _, err := h.Write(data); err != nil {
		return []byte{}, err
	}
	return h.Sum(nil), nil
}

// Upload stores a new data blob on the peer and will make it available to
// other peers. The blob will be split into chunks.
func (n *node) Upload(data io.Reader) (string, error) {
	// split the file into chunks
	// store the chunk, where the hash-code as the key to index it
	// store the metafile,
	chunks := make(map[string][]byte)
	chunkShaKeys := make([][]byte, 0, 10)
	chunkHexKeys := make([]string, 0, 10)

	chunkBuf := make([]byte, 0, n.conf.ChunkSize) // chunkBuf, which will be reused
	reachEOF := false
	budget := int(n.conf.ChunkSize)
	readBuf := make([]byte, budget)

	// assemble chunks, it should handle potential partial read
	for !reachEOF {
		nRead, err := data.Read(readBuf)
		chunkBuf = append(chunkBuf, readBuf[:nRead]...)

		if len(chunkBuf) == int(n.conf.ChunkSize) {
			chunk := make([]byte, len(chunkBuf))
			copy(chunk, chunkBuf)
			chunksha, err := sha256(chunk)
			chunkhash := hex.EncodeToString(chunksha)
			if err != nil {
				err = fmt.Errorf("Upload error: %w", err)
				n.Err(err).Send()
				return "", err
			}
			chunks[chunkhash] = chunk
			chunkShaKeys = append(chunkShaKeys, chunksha)
			chunkHexKeys = append(chunkHexKeys, chunkhash)

			// flush buffers
			budget = int(n.conf.ChunkSize)
			chunkBuf = chunkBuf[:0]
			readBuf = readBuf[:budget]
		} else {
			// flush readBuf and limit its budget to only fetch remaining part of a chunk
			budget -= nRead
			readBuf = readBuf[:budget]
		}

		// reach EOF, break loop
		if errors.Is(err, io.EOF) {
			// flush current valid chunk
			if len(chunkBuf) > 0 {
				chunk := make([]byte, len(chunkBuf))
				copy(chunk, chunkBuf)
				chunkBuf = chunkBuf[:0]
				chunksha, err := sha256(chunk)
				chunkhash := hex.EncodeToString(chunksha)
				if err != nil {
					err = fmt.Errorf("Upload error when hash chunk key: %w", err)
					n.Err(err).Send()
					return "", err
				}
				chunks[chunkhash] = chunk
				chunkShaKeys = append(chunkShaKeys, chunksha)
				chunkHexKeys = append(chunkHexKeys, chunkhash)

			}
			reachEOF = true
		} else if err != nil {
			err = fmt.Errorf("Upload error in read: %w", err)
			n.Err(err).Send()
			return "", err
		}

	}
	// we have assembled chunks, then we assemble the metafile
	metafileValue := strings.Join(chunkHexKeys, peer.MetafileSep)
	metafileKeySha, err := sha256(bytes.Join(chunkShaKeys, []byte{}))
	metafileKey := hex.EncodeToString(metafileKeySha)

	if err != nil {
		err = fmt.Errorf("Upload error when hash metafile key: %w", err)
		n.Err(err).Send()
		return "", err
	}

	// store chunks and metafile
	for key, value := range chunks {
		n.blob.Set(key, value)
	}
	n.blob.Set(metafileKey, []byte(metafileValue))

	return metafileKey, nil
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

	if !n.hasNeighbor() {
		n.Warn().Msg("no neighbor, cannot broadcast, direct return")
		return nil
	}

	ruMsg, err := n.msgRegistry.MarshalMessage(&types.RumorsMessage{Rumors: []types.Rumor{ru}})
	if err != nil {
		n.Err(err).Send()
		return fmt.Errorf("Broadcast error: %w", err)
	}

	// send and wait for the ack
	go func() {
		preNei := ""
		tried := 0
		acked := false
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
			if err != nil {
				n.Err(fmt.Errorf("Broadcast error: %w", err)).Send()
				// FIXME: this early return did not delete entries
				return
			}
			n.Debug().Str("dest", header.Destination).Str("nextDest", nextDest).Str("msg", ruMsg.String()).Str("pkt", pkt.String()).Msg("possibly sended")

			// start to wait for the ack message
			n.Debug().Msgf("start to wait for broadcast ack message on packet %s", pkt.Header.PacketID)
			select {
			case <-future:
				acked = true
				n.Debug().Msgf("ack received and properly processed")
			case <-time.After(n.conf.AckTimeout):
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

func (n *node) isKilled() bool {
	return atomic.LoadInt32(&n.stat) == KILL
}

func (n *node) addr() string {
	return n.sock.GetAddress()
}
