package impl

import (
	"bytes"
	"crypto"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.dedis.ch/cs438/logging"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/registry"
	"go.dedis.ch/cs438/storage"
	"go.dedis.ch/cs438/types"
)

var _logger zerolog.Logger = logging.RootLogger

// var _peerCount int32 = -1
var NONEIGHBOR string = "NONEIGHBOR"
var ErrNotFound error = errors.New("NOTFOUND")

// peer state
const (
	KILL = iota
	ALIVE
)

// func uniqueID() int32 {
// 	return atomic.AddInt32(&_peerCount, 1)
// }

// NewPeer creates a new peer. You can change the content and location of this
// function but you MUST NOT change its signature and package location.
func NewPeer(conf peer.Configuration) peer.Peer {
	// time.Now().Format()
	// here you must return a struct that implements the peer.Peer functions.
	// Therefore, you are free to rename and change it as you want.
	node := &node{msgRegistry: conf.MessageRegistry, id: int32(conf.PaxosID), conf: conf}
	node.blob = conf.Storage.GetDataBlobStore()
	node.naming = conf.Storage.GetNamingStore()
	node.catalog = peer.Catalog(make(map[string]map[string]struct{}))
	node.replyFutures = make(map[string]chan []byte)
	node.searchReplyFutures = make(map[string]chan *types.SearchReplyMessage)
	node.searchReqs = make(map[string]struct{})
	// init the routing table, add this.addr
	// node.seqs = make(map[string]uint)
	// node.rumors = make(map[string][]types.Rumor)
	// node.ackFutures = make(map[string]chan int)
	// node.route = peer.RoutingTable{node.addr(): node.addr()}
	// node.neighbors = make([]string, 0)
	// node.neighborSet = make(map[string]struct{})
	node.Messager = NewMessager(conf)
	node.consensus = NewConsensus(func(pv types.PaxosValue) error {
		node.naming.Set(pv.Filename, []byte(pv.Metahash))
		return nil
	}, node.Messager, conf)
	// node.Messaging = NewMessager(conf)

	node.chord = NewChord(node.Messager, conf)

	if node.conf.AckTimeout == 0 {
		node.conf.AckTimeout = math.MaxInt64
	}

	node.Logger = _logger.With().Str("Peer", fmt.Sprintf("%d %s", node.id, node.sock.GetAddress())).Logger()
	_logger.Info().Int32("id", node.id).Str("addr", node.sock.GetAddress()).Msg("create new peer")
	_logger.Debug().Msgf("conf=%s", conf.String())

	return node
}

// node implements a peer to build a Peerster system
//
// - implements peer.Peer
type node struct {
	zerolog.Logger

	// You probably want to keep the peer.Configuration on this struct:
	msgRegistry registry.Registry
	conf        peer.Configuration

	consensus Consensus

	chord *Chord
	// storage
	blob   storage.Store
	naming storage.Store

	cataMu  sync.Mutex
	catalog peer.Catalog

	// id          xid.ID
	// unique id, xid.ID is not human friendly
	id int32

	// TODO: these acks could be abstracted out
	// acuMu      sync.Mutex
	// ackFutures map[string]chan int

	replyMu      sync.Mutex
	replyFutures map[string]chan []byte

	searchReplyMu      sync.Mutex
	searchReplyFutures map[string]chan *types.SearchReplyMessage

	searchReqsMu sync.Mutex
	searchReqs   map[string]struct{}

	// messager
	*Messager // currently, we resort to simple composition
	// peer.Messaging // TODO: ultimately, we should make it interface

	stat int32
	// seqMu  sync.Mutex               // protect seqs and rumors
	// seqs   map[string]uint          // rumor seq of other nodes, here nodes are not necessarily the neighbors, since rumor corresponds to one origin
	// rumors map[string][]types.Rumor // key: node_addr value: rumors in increasing order of seq
}

// Start implements peer.Service
func (n *node) Start() error {
	n.Info().Msg("Starting...")
	n.stat = ALIVE
	n.msgRegistry.RegisterMessageCallback(types.ChatMessage{}, types.ChatMsgCallback)
	n.Trace().Msg("register callback for `ChatMessage`")
	n.msgRegistry.RegisterMessageCallback(types.RumorsMessage{}, n.RumorsMsgCallback)
	n.Trace().Msg("register callback for `RumorsMessage`")
	n.msgRegistry.RegisterMessageCallback(types.StatusMessage{}, n.StatusMsgCallback)
	n.Trace().Msg("register callback for `StatusMessage`")
	n.msgRegistry.RegisterMessageCallback(types.AckMessage{}, n.AckMsgCallback)
	n.Trace().Msg("register callback for `AckMessage`")
	n.msgRegistry.RegisterMessageCallback(types.EmptyMessage{}, types.EmptyMsgCallback)
	n.Trace().Msg("register callback for `EmptyMessage`")
	n.msgRegistry.RegisterMessageCallback(types.PrivateMessage{}, n.PrivateMsgCallback)
	n.Trace().Msg("register callback for `PrivateMessage`")
	n.msgRegistry.RegisterMessageCallback(types.DataRequestMessage{}, n.DataRequestMessageCallback)
	n.Trace().Msg("register callback for `DataRequestMessage`")
	n.msgRegistry.RegisterMessageCallback(types.DataReplyMessage{}, n.DataReplyMessageCallback)
	n.Trace().Msg("register callback for `DataReplyMessage`")
	n.msgRegistry.RegisterMessageCallback(types.SearchRequestMessage{}, n.SearchRequestMessageCallback)
	n.Trace().Msg("register callback for `SearchRequestMessage`")
	n.msgRegistry.RegisterMessageCallback(types.SearchReplyMessage{}, n.SearchReplyMessageCallback)
	n.Trace().Msg("register callback for `SearchReplyMessage`")
	// start a listining daemon to listen on the incoming message with `sock`

	n.Messager.Start()
	go n.stabilize(n.conf.StabilizeInterval)
	go n.fixFinger(n.conf.FixFingersInterval)
	n.Info().Msg("Start done")
	return nil
}

// Stop implements peer.Service
func (n *node) Stop() error {
	// panic("to be implemented in HW0")
	n.Info().Msg("Stoping...")
	atomic.StoreInt32(&n.stat, KILL)
	n.consensus.Stop()
	n.Messager.Stop()
	return nil
}

func (n *node) Store(key string) (err error) {
	n.chord.insertTable()
	n.chord.readTable()
	return nil
}

func (n *node) Join(member string) (err error) {
	return n.chord.join(member)
}

func (n *node) Init(member string) {
	n.chord.init(member)
}

func (n *node) Lookup(key string) (string, error) {
	fmt.Printf("hash key of %s is %d\n", key, n.chord.hashKey(key))
	destination, err := n.chord.findSuccessor(n.chord.hashKey(key))
	fmt.Printf("dest is %d\n", n.chord.hashKey(destination))
	return destination, err
}

func (n *node) LookupHashId(key uint) (uint, error) {
	destination, err := n.chord.findSuccessor(key)
	return n.chord.hashKey(destination), err
}

func (n *node) Get(key string) (string, bool) {
	// TODO: implemented when blockchain and smart contract finishes
	return "", true
}

func (n *node) Put(key string, data uint) {
	// TODO: implemented when blockchain and smart contract finishes
}

func (n *node) GetId(key uint) (uint, bool) {
	return n.chord.blockStore.get(key)
}

func (n *node) PutId(key uint, data uint) {
	n.chord.blockStore.put(key, data)
}

func (n *node) PrintInfo() uint {
	fmt.Println("--------------------------------------------------")
	fmt.Printf("Node %d: predecessor: %d, successor: %d\n",
		n.chord.hashKey(n.conf.Socket.GetAddress()),
		n.chord.hashKey(n.chord.predecessor.read()),
		n.chord.hashKey(n.chord.successor.read()))

	return n.chord.chordId
}

// for test
func (n *node) GetFingerTable() []uint {
	res := make([]uint, n.conf.ChordBits)
	for i := 0; i < int(n.conf.ChordBits); i++ {
		str, _ := n.chord.fingerTable.load(i)
		res[i] = n.chord.hashKey(str)
	}
	return res
}
func (n *node) GetChordId() uint {
	return n.chord.chordId
}
func (n *node) GetPredecessor() uint {
	return n.chord.hashKey(n.chord.predecessor.read())
}
func (n *node) GetSuccessor() uint {
	return n.chord.hashKey(n.chord.successor.read())
}

func (n *node) searchAllFromNei(reg regexp.Regexp, budget uint, timeout time.Duration) ([]string, error) {
	if !n.hasNeighbor() {
		return []string{}, nil
	}

	neis := n.getNeis()
	neis, budegts := budgetAllocation(neis, budget)
	n.Debug().Stack().Msgf("searchAll neis=%v, budgets=%v", neis, budegts)

	// now we have neis and associated budgets, we could send SearchRequest
	names := make([]string, 0, 10)
	nameSet := make(map[string]struct{})
	mu := sync.Mutex{}

	// allFinish := make(chan struct{}, 1)
	timer_ := time.After(timeout)
	timer := make(chan struct{}, 1) // this timer could signal multiple select
	go func() {
		<-timer_
		close(timer)
	}()
	var finishes sync.WaitGroup
	// ideally, we would receive #budget reponses
	// TODO: there would be edge cases, A<->B, budget=10, for example
	finishes.Add(int(budget))
	reqID := xid.New().String()
	n.searchReqsMu.Lock()
	n.searchReqs[reqID] = struct{}{} // itself knows this reqID
	n.searchReqsMu.Unlock()

	// construct a future
	future := n.searchReplyFuture(reqID)
	for i := range neis {
		nei, bud := neis[i], budegts[i]
		req_ := types.SearchRequestMessage{RequestID: reqID, Origin: n.addr(), Pattern: reg.String(), Budget: bud}
		if err := n.unicastTypesMsg(nei, &req_); err != nil {
			return []string{}, err
		}
	}
	// wait for the future in timeout period
	go func() {
		for {
			select {
			case reply := <-future:
				n.Warn().Msgf("req %s future received, %v", reply.RequestID, *reply)
				// append the return names
				for _, file := range reply.Responses {
					mu.Lock()
					nameSet[file.Name] = struct{}{}
					mu.Unlock()
				}
				finishes.Done()
			case <-timer:
				n.deleteSearchReplyFuture(reqID)
				n.Debug().Msg("timeout, end for loop of search reply receive")
				return
			}

		}
	}()
	// go func() {
	// 	// TODO: might hang forever?
	// 	// possible solutions:
	// 	// only use allFinish, and Done at each timeout in goroutine
	// 	finishes.Wait()
	// 	allFinish <- struct{}{}
	// }()

	// select {
	// case <-allFinish: // all get a result before the timeout
	// case <-time.After(timeout): // some still not return, but we should return the partial results
	// }
	allFinish := make(chan struct{})
	go func() {
		// TODO: it might hang
		finishes.Wait()
		allFinish <- struct{}{}
	}()
	select {
	case <-timer:
		n.Debug().Msg("timeout before all search reponses are received")

	case <-allFinish:
		n.Debug().Msg("all search reponses are received")
	}
	// after wait, following code is safe
	mu.Lock()
	for name := range nameSet {
		names = append(names, name)
	}
	mu.Unlock()
	return names, nil
}

func (n *node) searchFirstFromNei(reg regexp.Regexp, budget uint, timeout time.Duration) (string, error) {
	if !n.hasNeighbor() {
		return "", ErrNotFound
	}

	neis := n.getNeis()
	neis, budegts := budgetAllocation(neis, budget)
	n.Debug().Stack().Msgf("searchFirst neis=%v, budgets=%v", neis, budegts)

	// now we have neis and associated budgets, we could send SearchRequest
	fullKnownName := ""

	// allFinish := make(chan struct{}, 1)
	timer_ := time.After(timeout)
	timer := make(chan struct{}, 1) // this timer could signal multiple select
	go func() {
		<-timer_
		close(timer)
	}()

	// control channels
	findFullKnown := make(chan struct{}, 1)
	allFinish := make(chan struct{})

	// ideally, we would receive #budget reponses
	// TODO: there would be edge cases, A<->B, budget=10, for example
	var finishes sync.WaitGroup
	finishes.Add(int(budget))

	go func() {
		// TODO: it might hang
		finishes.Wait()
		allFinish <- struct{}{}
	}()

	reqID := xid.New().String()
	n.searchReqsMu.Lock()
	n.searchReqs[reqID] = struct{}{} // itself knows this reqID
	n.searchReqsMu.Unlock()

	// construct a future
	future := n.searchReplyFuture(reqID)
	for i := range neis {
		nei, bud := neis[i], budegts[i]
		req_ := types.SearchRequestMessage{RequestID: reqID, Origin: n.addr(), Pattern: reg.String(), Budget: bud}
		if err := n.unicastTypesMsg(nei, &req_); err != nil {
			return "", err
		}
	}
	// wait for the future in timeout period

	go func() {
		for {
			select {
			case reply := <-future:
				n.Warn().Msgf("searchFirst req %s future received, %v", reply.RequestID, *reply)
				// check if it is a fully matched file
				for _, file := range reply.Responses {
					fullKnown := true
					for _, chunk := range file.Chunks {
						if chunk == nil {
							fullKnown = false
							break
						}
					}
					if fullKnown {
						// we find the intended fully known file
						fullKnownName = file.Name
						findFullKnown <- struct{}{}
						finishes.Done()
						return
					}
				}
				finishes.Done()
			case <-timer:
				n.deleteSearchReplyFuture(reqID)
				n.Debug().Msg("timeout, end for loop of search reply receive")
				return
			}

		}
	}()

	select {
	case <-timer:
		n.Debug().Msg("searchFirst: timeout before all search reponses are received")
		return "", ErrNotFound
	case <-allFinish:
		n.Debug().Msg("searchFirst: all search reponses are received, but not find fullyKnownfile")
		return "", ErrNotFound
	case <-findFullKnown:
		n.Debug().Msgf("searchFirst find full knownfile: %s", fullKnownName)
		return fullKnownName, nil
	}
}

// SearchAll returns all the names that exist matching the given regex. It
// merges results from the local storage and from the search request reply
// sent to a random neighbor using the provided budget. It makes the peer
// update its catalog and name storage according to the SearchReplyMessages
// received. Returns an empty result if nothing found. An error is returned
// in case of an exceptional event.
// TODO: 这里返回的是 fileNames, 那么我们用哪个接口来 download file name 呢? 还是说必须得走一遍 naming store
// 那么 search callback 结果也要更新 naming store, 那么 searchReply 起码要返回 hashkey
// naming store 更新了还不够, catalog 也需要更新, 我们需要知道某个 hash 具体在哪个 peer, 我们有 hashkey 所以有这个信息了
func (n *node) SearchAll(reg regexp.Regexp, budget uint, timeout time.Duration) ([]string, error) {
	n.Info().Msgf("start search all req=%s, budget=%d, timeout=%v", reg.String(), budget, timeout)
	// 1. search local naming store
	matchNames := make([]string, 0, 10)
	n.naming.ForEach(func(key string, val []byte) bool {
		if reg.MatchString(key) {
			matchNames = append(matchNames, key)
		}
		return true
	})
	n.Info().Msgf("after local search, matchNames=%v", matchNames)
	// 2. search neighbors with budgets
	neiMatches, err := n.searchAllFromNei(reg, budget, timeout)
	if err != nil {
		err = fmt.Errorf("SearchAll partial fail on neighbor search: %w", err)
		n.Err(err).Send()
		return matchNames, err
	} else {
		matchNames = append(matchNames, neiMatches...)
	}
	n.Info().Msgf("after local+neighbor search, matchNames=%v", matchNames)

	// TODO: error logistics need to be re-checked
	return matchNames, nil
}

// SearchFirst uses an expanding ring configuration and returns a name as
// soon as it finds a peer that "fully matches" a data blob. It makes the
// peer update its catalog and name storage according to the
// SearchReplyMessages received. Returns an empty string if nothing was
// found.
func (n *node) SearchFirst(reg regexp.Regexp, conf peer.ExpandingRing) (name string, err error) {
	// first check if local store has a full version of the file

	n.Info().Msgf("start search first reg=%s, conf=%v", reg.String(), conf)
	// 1. search local naming store
	localMatched := false
	matchName := ""
	n.naming.ForEach(func(name string, metahash []byte) bool {
		if reg.MatchString(name) {
			// check if it is a full match
			// first fetch metafile
			if metafile := n.blob.Get(string(metahash)); metafile != nil {
				// parse the chunk keys, then fetch each content seperately
				chunkKeys := strings.Split(string(metafile), peer.MetafileSep)
				isFull := true
				for _, chunkKey := range chunkKeys {
					chunkValue := n.blob.Get(chunkKey)
					if chunkValue == nil {
						isFull = false
						break
					}
				}
				if isFull {
					matchName = name
					localMatched = true
					return false // stop traversal
				}
			}
		}
		return true
	})
	n.Info().Msgf("after local search, matchName=%v", matchName)
	if localMatched {
		return matchName, nil
	}

	// 2. search neighbors with budgets
	budget := conf.Initial
	for i := 0; i < int(conf.Retry); i++ {
		match, err := n.searchFirstFromNei(reg, budget, conf.Timeout)
		if err == nil {
			n.Info().Msgf("find match=%s from nei, return", match)
			return match, nil
		}
		if err != nil && errors.Is(err, ErrNotFound) {
			n.Info().Msgf("search first fail with budget=%d, retry=%d, try next", budget, i)
			budget *= conf.Factor
			continue
		}
		if err != nil {
			// exceptional error
			err = fmt.Errorf("SearchFirst error: %w", err)
			n.Err(err).Send()
			return "", err
		}
	}
	// if we did not find, it is not an err
	n.Info().Msgf("search first fail on all retries, return")
	return "", nil

}

// Tag creates a mapping between a (file)name and a metahash.
//
func (n *node) Tag(name string, mh string) error {
	__logger := n.Logger.With().Str("func", "Tag").Logger()

	if n.conf.TotalPeers <= 1 {
		n.naming.Set(name, []byte(mh))
		return nil
	}
	__logger.Info().Msgf("try to tag on name=%s, mh=%s", name, mh)
	value := types.PaxosValue{UniqID: xid.New().String(), Filename: name, Metahash: mh}

	for {
		if n.naming.Get(name) != nil {
			return fmt.Errorf("filename=%s already exists in naming store", name)
		}
		__logger.Info().Msgf("name=%s not in namestore yet", name)

		__logger.Info().Msgf("try to tag and propose on value=%v", value)
		finish, err := n.consensus.Propose(value)
		if errors.Is(err, ErrOnProposing) {
			__logger.Info().Msgf("it is onProposing, we need to wait for finish")
			<-finish // wait to be done
			__logger.Info().Msgf("finished, could start another check")

		} else {
			__logger.Info().Msgf("we are now proposing")
			consensuValue := <-finish
			__logger.Info().Msgf("reach consensus on value=%v, while our tag vakue=%v", consensuValue, value)
			if value.UniqID == consensuValue.UniqID {
				__logger.Info().Msgf("consensusValue=ourValue, return well", consensuValue, value)
				return nil
			}
			__logger.Info().Msgf("consensusValue!=ourValue, start another check", consensuValue, value)

		}

	}
}

// Resolve returns the corresponding metahash of a given (file)name. Returns
// an empty string if not found.
func (n *node) Resolve(name string) string {
	metahash := n.naming.Get(name)
	return string(metahash)
}

func sha256(data []byte) ([]byte, error) {
	h := crypto.SHA256.New()
	if _, err := h.Write(data); err != nil {
		return []byte{}, err
	}
	return h.Sum(nil), nil
}

// return a copy
func (n *node) GetCatalog() peer.Catalog {
	n.cataMu.Lock()
	defer n.cataMu.Unlock()
	ret := make(map[string]map[string]struct{})
	for k, v := range n.catalog {
		copyV := make(map[string]struct{})
		for peer := range v {
			copyV[peer] = struct{}{}
		}
		ret[k] = copyV
	}
	return ret
}

func (n *node) UpdateCatalog(key string, peer string) {
	// TODO: is it thread-safe?
	n.cataMu.Lock()
	defer n.cataMu.Unlock()
	n.Info().Msgf("add key=%s, peer=%s in catalog", key, peer)
	if peers, ok := n.catalog[key]; ok {
		peers[peer] = struct{}{}
	} else {
		n.catalog[key] = make(map[string]struct{})
		n.catalog[key][peer] = struct{}{}
	}
}

func (n *node) searchReplyFuture(reqID string) chan *types.SearchReplyMessage {
	ret := make(chan *types.SearchReplyMessage, 1)
	n.searchReplyMu.Lock()
	n.searchReplyFutures[reqID] = ret
	n.searchReplyMu.Unlock()
	return ret
}

func (n *node) deleteSearchReplyFuture(reqID string) {
	n.searchReplyMu.Lock()
	delete(n.searchReplyFutures, reqID)
	n.searchReplyMu.Unlock()
}

func (n *node) dataReplyFuture(reqID string) chan []byte {
	ret := make(chan []byte, 1)
	n.replyMu.Lock()
	n.replyFutures[reqID] = ret
	n.replyMu.Unlock()
	return ret
}

func (n *node) deleteReplyFuture(reqID string) {
	n.replyMu.Lock()
	delete(n.replyFutures, reqID)
	n.replyMu.Unlock()
}

func (n *node) download(hexhash string) ([]byte, error) {
	if content := n.blob.Get(hexhash); content != nil {
		return content, nil
	}

	n.cataMu.Lock()
	peersSet, ok := n.catalog[hexhash]
	n.cataMu.Unlock()

	if ok {
		// select a random peer and send DataRequestMessage
		peers := make([]string, 0, len(peersSet))
		for k := range peersSet {
			peers = append(peers, k)
		}
		peer := peers[rand.Int63n(int64(len(peers)))]

		// back off strategy
		waitTime := n.conf.BackoffDataRequest.Initial
		for i := 0; i < int(n.conf.BackoffDataRequest.Retry); i++ {
			// TODO: shall we use different reqID? or we should use same reqID?
			reqID := xid.New().String()
			msg_ := types.DataRequestMessage{RequestID: reqID, Key: hexhash}
			msg, err := n.msgRegistry.MarshalMessage(&msg_)

			if err != nil {
				return nil, err
			}

			replyFuture := n.dataReplyFuture(reqID)
			if err = n.Unicast(peer, msg); err != nil {
				n.deleteReplyFuture(reqID)
				return nil, err
			}

			// wait for the DataReplyMessage
			select {
			case <-time.After(waitTime):
				// wait for next backoff time, what about reqID?
				n.Info().Msgf("waitTime=%v elapsed, backoff for %s", waitTime, reqID)
			case content := <-replyFuture:
				if len(content) == 0 {
					return nil, fmt.Errorf("neighbor %s does not have %s", peer, hexhash)
				}
				n.Info().Msgf("receive content for req %s", reqID)
				return content, nil
			}
			// delete registered future, to avoid infinite growth of the map
			n.deleteReplyFuture(reqID)
			waitTime *= time.Duration(n.conf.BackoffDataRequest.Factor)
		}

		return nil, fmt.Errorf("backoff timeout")

	}
	return nil, fmt.Errorf("no one has the file")
}

// Download will get all the necessary chunks corresponding to the given
// metahash that references a blob, and return a reconstructed blob. The
// peer will save locally the chunks that it doesn't have for further
// sharing. Returns an error if it can't get the necessary chunks.
func (n *node) Download(metahash string) ([]byte, error) {
	// 1. fetch metahash
	metaContent, err := n.download(metahash)
	if err != nil {
		err = fmt.Errorf("Download error: %w", err)
		n.Err(err).Send()
		return nil, err
	}
	// 2. parse chunks metahash key from metafile and fetch the chunks
	chunkHexHashs := strings.Split(string(metaContent), peer.MetafileSep)
	chunks := make([][]byte, len(chunkHexHashs))
	for _, chunkHexHash := range chunkHexHashs {
		chunkContent, err := n.download(chunkHexHash)
		if err != nil {
			err = fmt.Errorf("Download error: %w", err)
			n.Err(err).Send()
			return nil, err
		}
		chunks = append(chunks, chunkContent)
	}

	return bytes.Join(chunks, []byte{}), nil
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

func (n *node) isKilled() bool {
	return atomic.LoadInt32(&n.stat) == KILL
}

func (n *node) addr() string {
	return n.sock.GetAddress()
}
