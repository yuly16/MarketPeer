package impl

import (
	"crypto/sha1"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/registry"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"math/big"
	"sync"
	"time"
)





func NewChord(messager peer.Messaging, conf peer.Configuration) *Chord {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)

	chordInstance := Chord{}
	chordInstance.conf = conf
	chordInstance.msgRegistry = conf.MessageRegistry
	chordInstance.Messaging = messager

	chordInstance.predecessor = MutexString{data: ""}
	chordInstance.successor = MutexString{data: ""}

	chordId := chordInstance.hashKey(conf.Socket.GetAddress())
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

type ChordStorage struct {
	sync.Mutex
	data map[uint]uint
}

func (c *ChordStorage) get(key uint) (uint, bool) {
	c.Lock()
	defer c.Unlock()
	data, ok := c.data[key]
	return data, ok
}

func (c *ChordStorage) put(key uint, data uint) {
	c.Lock()
	defer c.Unlock()
	c.data[key] = data
}

//func (c *ChordStorage)
type Chord struct {
	peer.Messaging
	zerolog.Logger

	conf                peer.Configuration
	msgRegistry         registry.Registry

	chordId             uint
	successor           MutexString
	predecessor         MutexString

	chMutex             sync.Mutex
	findSuccessorCh     map[uint]chan types.ChordFindSuccessorReplyMessage
	askPredecessorCh    chan types.ChordReplyPredecessorMessage
	fingerTable         *FingerTable

	blockStore          ChordStorage
}

//The identifier length m must
//be large enough to make the probability of two nodes or keys
//hashing to the same identifier negligible.

func (c *Chord) hashKey(key string) uint {
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
		if between(c.hashKey(nodeAddr), c.chordId, id) {
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
			betweenRightInclude(id, c.hashKey(predecessor), c.chordId)) {
		return c.conf.Socket.GetAddress(), nil
	}
	// case 1: if successor = "". This case happens in the initialization of chord
	if successor == "" {
		return c.conf.Socket.GetAddress(), nil
	}

	// case 2: if id is in (chordId, successor], return the address of successor
	if betweenRightInclude(id, c.chordId, c.hashKey(successor)) {
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
		c.chordId, c.hashKey(dest), id)

	// waiting for successor of id
	timer := time.After(500 * time.Second)
	select {
	case findSuccMsg := <- findSuccCh:
		log.Debug().Msgf("FindSuccessorRemote: %d receives successorReply from %d, id = %d\n",
			c.chordId, c.hashKey(dest), id)
		return findSuccMsg.Dest, nil
	case <-timer:
		return "", fmt.Errorf("FindSuccessorRemote: waiting for successor time out. ")
	}
}

func (c *Chord) join(member string) error {
	if successor, err := c.FindSuccessorRemote(member, c.chordId); err == nil {
		if successor != c.conf.Socket.GetAddress() {
			c.successor.write(successor)
		}
		return nil
	} else {
		return err
	}
}

func (c *Chord) init(member string) {
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
			between(c.hashKey(ReplyPredecessorMsg.Predecessor), c.chordId, c.hashKey(successor)) {
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


func (c *Chord) ChordFindSuccessorCallback(msg types.Message, pkt transport.Packet) error {
	successor := c.successor.read()
	predecessor := c.predecessor.read()
	findSuccessorMsg := msg.(*types.ChordFindSuccessorMessage)
	log.Debug().Msgf("ChordFindSuccessorCallback: %d receives findSuccessor of %d, id = %d\n",
		c.chordId, c.hashKey(findSuccessorMsg.Source), findSuccessorMsg.ID)
	if successor == c.conf.Socket.GetAddress() {
		return fmt.Errorf("ChordFindSuccessorCallback: successor is equal to currnode! ")
	}
	// case 0: if id is in (predecessor, chordId], return current node
	// refer to https://www.kth.se/social/upload/51647996f276545db53654c0/3-chord.pdf page 22
	// FIXME: not sure about this. The paper doesn't mention it
	if findSuccessorMsg.ID == c.chordId ||
		(predecessor != "" &&
			betweenRightInclude(findSuccessorMsg.ID, c.hashKey(predecessor), c.chordId)) {
		msg, err := c.msgRegistry.MarshalMessage(
			types.ChordFindSuccessorReplyMessage{Dest: c.conf.Socket.GetAddress(), ID: findSuccessorMsg.ID})
		if err != nil {
			return err
		}
		errUnicast := c.Messaging.Unicast(findSuccessorMsg.Source, msg)
		if errUnicast != nil {
			return errUnicast
		}
	}

	// case 1: if successor = "". This case happens in the initialization of chord
	if successor == "" {
		msg, err := c.msgRegistry.MarshalMessage(
			types.ChordFindSuccessorReplyMessage{Dest: c.conf.Socket.GetAddress(), ID: findSuccessorMsg.ID})
		if err != nil {
			return err
		}
		//log.Debug().Msgf("ChordFindSuccessorCallback: case 1 : Successor = nil. " +
		//	"%d sends findSuccessorReply to %d, id = %d\n",
		//	c.chordId, c.hashKey(findSuccessorMsg.Source), findSuccessorMsg.ID)
		errUnicast := c.Messaging.Unicast(findSuccessorMsg.Source, msg)
		if errUnicast != nil {
			return errUnicast
		}
	// case 2: if id is in (chordId, successor], return the address of successor
	} else if betweenRightInclude(findSuccessorMsg.ID, c.chordId, c.hashKey(successor)) {
		msg, err := c.msgRegistry.MarshalMessage(
			types.ChordFindSuccessorReplyMessage{Dest: successor, ID: findSuccessorMsg.ID})
		if err != nil {
			return err
		}
		log.Debug().Msgf("ChordFindSuccessorCallback: case 2 Successor = %d. " +
			"%d sends findSuccessorReply to %d, id = %d\n",
			c.hashKey(successor), c.chordId, findSuccessorMsg.Source, findSuccessorMsg.ID)
		errUnicast := c.Messaging.Unicast(findSuccessorMsg.Source, msg)
		if errUnicast != nil {
			return errUnicast
		}
	} else {
		nStar, err1 := c.closestPrecedingNode(findSuccessorMsg.ID)
		if err1 != nil {
			return err1
		}

		if nStar == c.conf.Socket.GetAddress() {
			msg, err := c.msgRegistry.MarshalMessage(
				types.ChordFindSuccessorReplyMessage{Dest: successor, ID: findSuccessorMsg.ID})
			if err != nil {
				return err
			}
			log.Debug().Msgf("ChordFindSuccessorCallback: case 3.1 Successor = %d. " +
				"%d sends findSuccessorReply to %d, id = %d\n",
				c.hashKey(successor), c.chordId, findSuccessorMsg.Source, findSuccessorMsg.ID)
			errUnicast := c.Messaging.Unicast(findSuccessorMsg.Source, msg)
			if errUnicast != nil {
				return errUnicast
			}
		} else {
			log.Debug().Msgf("ChordFindSuccessorCallback: case 3.2 Source = %d. " +
				"%d relay to %d, id = %d\n",
				c.hashKey(findSuccessorMsg.Source), c.chordId, c.hashKey(nStar), findSuccessorMsg.ID)
			if err := c.RequestSuccessorRemote(findSuccessorMsg.Source,
				nStar, findSuccessorMsg.ID); err != nil {
				return err
			}
		}

	}
	return nil
}

func (c *Chord) ChordFindSuccessorReplyCallback(msg types.Message, pkt transport.Packet) error {
	findSuccessorReplyMsg := msg.(*types.ChordFindSuccessorReplyMessage)
	c.chMutex.Lock()
	if ch, ok := c.findSuccessorCh[findSuccessorReplyMsg.ID]; ok {
		ch <- *findSuccessorReplyMsg
	} else {
		return fmt.Errorf("ChordFindSuccessorReplyCallback: the channel doesn't exist. ")
	}
	c.chMutex.Unlock()

	return nil
}

func (c *Chord) ChordAskPredecessorCallback(msg types.Message, pkt transport.Packet) error {
	replyMsg, err := c.msgRegistry.MarshalMessage(
		types.ChordReplyPredecessorMessage{Predecessor: c.predecessor.read()})
	if err != nil {
		return err
	}
	if errUnicast := c.Messaging.Unicast(pkt.Header.Source, replyMsg); errUnicast != nil {
		return errUnicast
	}
	return nil
}

func (c *Chord) ChordReplyPredecessorCallback(msg types.Message, pkt transport.Packet) error {
	replyMsg := msg.(*types.ChordReplyPredecessorMessage)
	c.askPredecessorCh <- *replyMsg
	return nil
}

func (c *Chord) ChordNotifyCallback(msg types.Message, pkt transport.Packet) error {
	nStar := pkt.Header.Source
	nStarId := c.hashKey(nStar)
	predecessor := c.predecessor.read()
	if predecessor == "" ||
		between(nStarId, c.hashKey(predecessor), c.chordId) {
		c.predecessor.write(nStar)
		//log.Debug().Msgf("NotifyCallback: %d receives %d as predecessor\n", c.chordId, nStarId)
	}
	return nil
}

func (c *Chord) print() {
	id := c.hashKey(c.conf.Socket.GetAddress())
	fmt.Printf("caonima %d\n", id)
}

func (c *Chord) insertTable() {
	c.fingerTable.insert(0, c.conf.Socket.GetAddress())
}

func (c *Chord) readTable() {
	aaa,err:= c.fingerTable.load(4)
	fmt.Println(len(aaa))
	fmt.Println(err)
	fmt.Println(aaa)
}

