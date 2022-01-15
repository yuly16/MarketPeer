package chord

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)


func (c *Chord) ChordFindSuccessorCallback(msg types.Message, pkt transport.Packet) error {
	successor := c.successor.read()
	predecessor := c.predecessor.read()
	findSuccessorMsg := msg.(*types.ChordFindSuccessorMessage)
	log.Debug().Msgf("ChordFindSuccessorCallback: %d receives findSuccessor of %d, id = %d\n",
		c.chordId, c.HashKey(findSuccessorMsg.Source), findSuccessorMsg.ID)
	if successor == c.conf.Socket.GetAddress() {
		return fmt.Errorf("ChordFindSuccessorCallback: successor is equal to currnode! ")
	}
	// case 0: if id is in (predecessor, chordId], return current node
	// refer to https://www.kth.se/social/upload/51647996f276545db53654c0/3-chord.pdf page 22
	// FIXME: not sure about this. The paper doesn't mention it
	if findSuccessorMsg.ID == c.chordId ||
		(predecessor != "" &&
			betweenRightInclude(findSuccessorMsg.ID, c.HashKey(predecessor), c.chordId)) {
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
	} else if betweenRightInclude(findSuccessorMsg.ID, c.chordId, c.HashKey(successor)) {
		msg, err := c.msgRegistry.MarshalMessage(
			types.ChordFindSuccessorReplyMessage{Dest: successor, ID: findSuccessorMsg.ID})
		if err != nil {
			return err
		}
		log.Debug().Msgf("ChordFindSuccessorCallback: case 2 Successor = %d. " +
			"%d sends findSuccessorReply to %d, id = %d\n",
			c.HashKey(successor), c.chordId, findSuccessorMsg.Source, findSuccessorMsg.ID)
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
				c.HashKey(successor), c.chordId, findSuccessorMsg.Source, findSuccessorMsg.ID)
			errUnicast := c.Messaging.Unicast(findSuccessorMsg.Source, msg)
			if errUnicast != nil {
				return errUnicast
			}
		} else {
			log.Debug().Msgf("ChordFindSuccessorCallback: case 3.2 Source = %d. " +
				"%d relay to %d, id = %d\n",
				c.HashKey(findSuccessorMsg.Source), c.chordId, c.HashKey(nStar), findSuccessorMsg.ID)
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
	nStarId := c.HashKey(nStar)
	predecessor := c.predecessor.read()
	if predecessor == "" ||
		between(nStarId, c.HashKey(predecessor), c.chordId) {
		c.predecessor.write(nStar)
		//log.Debug().Msgf("NotifyCallback: %d receives %d as predecessor\n", c.chordId, nStarId)
	}
	return nil
}


