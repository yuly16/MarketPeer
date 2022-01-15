package chord

import (
	"crypto/sha1"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/registry"
	"go.dedis.ch/cs438/types"
	"math/big"
	"sync"
	"sync/atomic"
	"time"
)





func NewChord(messager peer.Messaging, conf peer.Configuration) *Chord {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	chordInstance := Chord{}
	chordInstance.conf = conf
	chordInstance.msgRegistry = conf.MessageRegistry
	chordInstance.Messaging = messager

	chordInstance.predecessor = MutexString{data: ""}
	chordInstance.successor = MutexString{data: ""}

	chordId := chordInstance.HashKey(conf.Socket.GetAddress())
	fmt.Printf("%s: %d\n", conf.Socket.GetAddress(), chordId)
	chordInstance.chordId = chordId

	chordInstance.fingerTable = NewFingerTable(conf)

	chordInstance.msgRegistry.RegisterMessageCallback(types.ChordFindSuccessorMessage{}, chordInstance.ChordFindSuccessorCallback)
	chordInstance.msgRegistry.RegisterMessageCallback(types.ChordFindSuccessorReplyMessage{}, chordInstance.ChordFindSuccessorReplyCallback)
	chordInstance.msgRegistry.RegisterMessageCallback(types.ChordAskPredecessorMessage{}, chordInstance.ChordAskPredecessorCallback)
	chordInstance.msgRegistry.RegisterMessageCallback(types.ChordReplyPredecessorMessage{}, chordInstance.ChordReplyPredecessorCallback)
	chordInstance.msgRegistry.RegisterMessageCallback(types.ChordNotifyMessage{}, chordInstance.ChordNotifyCallback)

	chordInstance.findSuccessorCh = make(map[uint]chan types.ChordFindSuccessorReplyMessage)
	chordInstance.askPredecessorCh = make(chan types.ChordReplyPredecessorMessage, 1)

	chordInstance.blockStore.data = make(map[uint]uint)

	return &chordInstance
}


//func (c *ChordStorage)
type Chord struct {
	peer.Messaging
	zerolog.Logger

	conf                peer.Configuration
	msgRegistry         registry.Registry

	chordId     		uint
	successor   		MutexString
	predecessor 		MutexString

	chMutex             sync.Mutex
	findSuccessorCh     map[uint]chan types.ChordFindSuccessorReplyMessage
	askPredecessorCh    chan types.ChordReplyPredecessorMessage
	fingerTable         *FingerTable

	blockStore          ChordStorage

	stat                int32
}

//The identifier length m must
//be large enough to make the probability of two nodes or keys
//hashing to the same identifier negligible.

func (c *Chord) HashKey(key string) uint {
	h := sha1.New()
	if _, err := h.Write([]byte(key)); err != nil {
		return 0
	}
	val := h.Sum(nil)
	valInt := (&big.Int{}).SetBytes(val)

	return uint(valInt.Uint64()) % (1 << c.conf.ChordBits)
}

func (c *Chord) closestPrecedingNode(id uint) (string, error) {
	for i := int(c.conf.ChordBits - 1); i >= 0; i -- {
		nodeAddr, err := c.fingerTable.load(i)
		if err != nil {
			return "", err
		}
		if nodeAddr == "" {
			continue
		}
		if between(c.HashKey(nodeAddr), c.chordId, id) {
			return nodeAddr, nil
		}
	}
	return c.conf.Socket.GetAddress(), nil
}

// find the successor of id, and return the address of successor
func (c *Chord) findSuccessor(id uint) (string, error) {
	successor := c.successor.read()
	predecessor := c.predecessor.read()
	// case 0: if id is in (predecessor, chordId], return current node
	// refer to https://www.kth.se/social/upload/51647996f276545db53654c0/3-chord.pdf page 22
	// FIXME: not sure about this. The paper doesn't mention it
	if id == c.chordId ||
		(predecessor != "" &&
			betweenRightInclude(id, c.HashKey(predecessor), c.chordId)) {
		return c.conf.Socket.GetAddress(), nil
	}
	// case 1: if successor = "". This case happens in the initialization of chord
	if successor == "" {
		return c.conf.Socket.GetAddress(), nil
	}

	// case 2: if id is in (chordId, successor], return the address of successor
	if betweenRightInclude(id, c.chordId, c.HashKey(successor)) {
		return successor, nil
	}

	// case 3: if id is not in (chordId, successor]: call closestPredecessor to find the successor
	nStar, err := c.closestPrecedingNode(id)
	if err != nil {
		return "", err
	}
	// case 3.1 if nStar is equal to current node: return the successor of current node
	// note: successor can't be nil! see the first line of this function
	if nStar == c.conf.Socket.GetAddress() {
		return successor, nil
	}
	// case 3.2 if nStar is not equal to current node:
	successorOfId, err := c.FindSuccessorRemote(nStar, id)
	if err != nil {
		return "", err
	}
	return successorOfId, nil

}

// RequestSuccessorRemote tells dest to find the Successor of id and send to source
func (c *Chord) RequestSuccessorRemote(source string, dest string, id uint) error {
	msg, err := c.msgRegistry.MarshalMessage(
		types.ChordFindSuccessorMessage{Source: source, ID:id})
	if err != nil {
		return err
	}
	errUnicast := c.Unicast(dest, msg)
	if errUnicast != nil {
		return errUnicast
	}
	return nil
}

// FindSuccessorRemote tells dest to find successor of id. waiting for the response of dest,
// and return the address of successor.
func (c *Chord) FindSuccessorRemote(dest string, id uint) (string, error) {
	// create a channel
	var findSuccCh chan types.ChordFindSuccessorReplyMessage

	c.chMutex.Lock()
	if ch, ok := c.findSuccessorCh[id]; ok {
		findSuccCh = ch
	} else {
		ch := make(chan types.ChordFindSuccessorReplyMessage, 10)
		findSuccCh = ch
		c.findSuccessorCh[id] = findSuccCh
	}
	c.chMutex.Unlock()

	if err := c.RequestSuccessorRemote(c.conf.Socket.GetAddress(), dest, id); err != nil {
		return "", err
	}
	log.Debug().Msgf("FindSuccessorRemote: %d waits successorReply, send to %d, id = %d\n",
		c.chordId, c.HashKey(dest), id)

	// waiting for successor of id
	timer := time.After(500 * time.Second)
	select {
	case findSuccMsg := <- findSuccCh:
		log.Debug().Msgf("FindSuccessorRemote: %d receives successorReply from %d, id = %d\n",
			c.chordId, c.HashKey(dest), id)
		return findSuccMsg.Dest, nil
	case <-timer:
		return "", fmt.Errorf("FindSuccessorRemote: waiting for successor time out. ")
	}
}

func (c *Chord) Join(member string) error {
	if successor, err := c.FindSuccessorRemote(member, c.chordId); err == nil {
		if successor != c.conf.Socket.GetAddress() {
			c.successor.write(successor)
		}
		return nil
	} else {
		return err
	}
}

func (c *Chord) Init(member string) {
	if c.successor.read() == "" && c.predecessor.read() == "" {
		c.successor.write(member)
		c.predecessor.write(member)
	}
}
func (c *Chord) Stabilize() error {
	successor := c.successor.read()
	if successor == "" {
		return nil
	}
	// ask predecessor to its successor
	msg, err := c.msgRegistry.MarshalMessage(
		types.ChordAskPredecessorMessage{})
	if err != nil {
		return err
	}
	errUnicast := c.Unicast(successor, msg)
	if errUnicast != nil {
		return errUnicast
	}

	// waiting for the response of successor
	timer := time.After(3 * time.Second)
	select {
	case ReplyPredecessorMsg := <- c.askPredecessorCh:
		if ReplyPredecessorMsg.Predecessor != "" &&
			between(c.HashKey(ReplyPredecessorMsg.Predecessor), c.chordId, c.HashKey(successor)) {
			c.successor.write(ReplyPredecessorMsg.Predecessor)
			//log.Debug().Msgf("Stabilize: %d receives successor %d's predecessor %d as new successor\n",
			//	c.chordId, c.hashKey(successor), c.hashKey(ReplyPredecessorMsg.Predecessor))
		}
		// need to refresh successor!
		newSuccessor := c.successor.read()
		// send notifyMsg to successor
		msg, err := c.msgRegistry.MarshalMessage(
			types.ChordNotifyMessage{})
		if err != nil {
			return err
		}
		errUnicast := c.Unicast(newSuccessor, msg)
		if errUnicast != nil {
			return errUnicast
		}
		return nil
	case <-timer:
		return fmt.Errorf("Stabilize: waiting for successor time out. ")
	}
}

func (c *Chord) FixFinger() error {
	fingerTableItem, err := c.findSuccessor((c.chordId + 1 << c.fingerTable.fixPointer) % (1 << c.conf.ChordBits))
	if err != nil {
		return err
	}
	if fingerTableItem != "" {
		err1 := c.fingerTable.insert(int(c.fingerTable.fixPointer), fingerTableItem)
		if err1 != nil {
			return err1
		}
		c.fingerTable.fixPointer = (c.fingerTable.fixPointer + 1) % c.conf.ChordBits
	}
	return nil
}

func (c *Chord) Lookup(key uint) (string, error) {
	destination, err := c.findSuccessor(key)
	return destination, err
}


func (c *Chord) Get(key uint) (uint, bool) {
	return c.blockStore.get(key)
}

func (c *Chord) Put(key uint, data uint) {
	c.blockStore.put(key, data)
}

func (c *Chord) GetChordId() uint{
	return c.chordId
}

func (c *Chord) GetPredecessor() string {
	return c.predecessor.read()
}

func (c *Chord) GetSuccessor() string {
	return c.successor.read()
}

// just for test
func (c *Chord) GetFingerTableItem(i int) (string, error) {
	return c.fingerTable.load(i)
}

func (c *Chord) GetFingerTable() []uint {
	res := make([]uint, c.conf.ChordBits)
	for i := 0; i < int(c.conf.ChordBits); i++ {
		str, _ := c.GetFingerTableItem(i)
		res[i] = c.HashKey(str)
	}
	return res
}


func (c *Chord) isKilled() bool {
	return atomic.LoadInt32(&c.stat) == KILL
}

func (c *Chord) Stop() {
	atomic.StoreInt32(&c.stat, KILL)
}

func (c *Chord) Start() {
	go c.stabilizeDaemon(c.conf.StabilizeInterval)
	go c.fixFingerDaemon(c.conf.FixFingersInterval)
	atomic.StoreInt32(&c.stat, ALIVE)
}