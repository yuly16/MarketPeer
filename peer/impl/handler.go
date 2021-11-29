package impl

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

// assumption: only ListenDaemon could exec RumorsMsgCallback;
// Q: possible race condition on packet?
// A: no, only possible inplace modificiation of packet happens after RumorMsgCallback processed.
// seqs 讨论: 可能的 race 来自于 BroadCast 对 self.seqID 的更新
// rumor process 讨论: 每一个 origin 的 rumor 一定是按序 process 的, 并且不会有任何的 race
// rumor: it only embed higher-level message. It will not embed rumors, status, ack etc.

// Q: seqs update shall onReceived or onProcessed?
// A: currently onReceived. Because received req is guaranteed to be processed eventually.
func (n *Messager) RumorsMsgCallback(msg types.Message, pkt transport.Packet) error {
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
		pkt_ := transport.Packet{
			Header: pkt.Header,
			Msg:    rumor.Msg,
		}
		__logger.Info().Msg("now processing embed message")
		if err := n.msgRegistry.ProcessPacket(pkt_); err != nil {
			__logger.Err(err).Send()
			continue
			// return fmt.Errorf("RumorsMsgCallback fail: processing rumor's embed msg fail: %w", err)
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

	// 3. possibly redirect the RumorsMessage to another Messager incase it is "new"
	// for those msgs that are ignored, are they not new. see https://moodle.epfl.ch/mod/forum/discuss.php?d=65056
	if !isNew {
		__logger.Info().Msg("nothing new from rumors, dont need to propagate, return")
		return nil
	}

	// here we create a new packet and use this Messager as source, since this is a re-send rather than routing
	// dont need to check neighbor, otherwise we cannot receive this message.
	// TODO: it might be not right
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
func (n *Messager) StatusMsgCallback(msg types.Message, pkt transport.Packet) error {
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

	} else if otherExceptMe {
		// send StatusMessage to other, such that other would send my-missing rumors back to me
		if marshalErr != nil {
			n.Err(marshalErr).Send()
			return fmt.Errorf("StatusMsgCallback fail: send status message back: %w", marshalErr)
		}
		n.Info().Msgf("otherExceptMe valid, unicast statusMsg %s to %s", possibleStatusMsg, other)
		if err := n.Unicast(other, possibleStatusMsg); err != nil {
			n.Err(err).Send()
			return fmt.Errorf("StatusMsgCallback fail: send status message back: %w", err)
		}
	}

	// me and other has exactly the same view
	if len(meExceptOther) == 0 && !otherExceptMe && rand.Float64() < n.conf.ContinueMongering {
		// "ContinueMongering"
		// send to a random nei other than other
		// Note: dont need to check neight existence, since we receive a status from a connected Messager
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
		n.Logger.Info().Msgf("continue mongering to neighbor %s with status=%s", nei, possibleStatus)
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
func (n *Messager) AckMsgCallback(msg types.Message, pkt transport.Packet) error {
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

	// FIXME: too many Marshalcall and error checking
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

func (n *Messager) PrivateMsgCallback(msg types.Message, pkt transport.Packet) error {
	private := msg.(*types.PrivateMessage)
	n.Info().Msgf("enter private callback, private=%s", private)
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

func (n *node) DataReplyMessageCallback(msg types.Message, pkt transport.Packet) error {
	n.Debug().Msg("start data reply callback")
	reply := msg.(*types.DataReplyMessage)
	n.replyMu.Lock()
	future, ok := n.replyFutures[reply.RequestID]
	n.replyMu.Unlock()

	// store the element to its local blob storage
	// TODO: what if the future ok=false? or at some edge cases? it's a late reply case
	n.blob.Set(reply.Key, reply.Value)

	if ok {
		n.Info().Msgf("notify reply future %s", reply.RequestID)

		future <- reply.Value
	} else {
		// do nothing, it is a arrive-late ack msg
		n.Info().Msgf("data reply(%s) packet %s has no future to complete", reply.RequestID, pkt.Header.PacketID)
	}

	return nil
}

func (n *node) DataRequestMessageCallback(msg types.Message, pkt transport.Packet) error {
	n.Debug().Msg("start data req callback")
	req := msg.(*types.DataRequestMessage)
	reply_ := types.DataReplyMessage{Key: req.Key, RequestID: req.RequestID, Value: n.blob.Get(req.Key)}
	reply, err := n.msgRegistry.MarshalMessage(&reply_)
	if err != nil {
		n.Err(err).Send()
		return err
	}
	err = n.strictUnicast(pkt.Header.Source, reply)
	if err != nil {
		n.Err(err).Send()
		return err
	}

	return nil
}

func (n *node) SearchReplyMessageCallback(msg types.Message, pkt transport.Packet) error {
	n.Warn().Msg("start search reply callback")
	reply := msg.(*types.SearchReplyMessage)
	n.searchReplyMu.Lock()
	future, ok := n.searchReplyFutures[reply.RequestID]
	n.searchReplyMu.Unlock()

	// Note: we should do the update before signal the future. otherwise tester
	// might use GetCataLog to get a stale result.
	// store the element to its local blob storage
	// update naming store
	// update catalog
	// TODO: what if the future ok=false? or at some edge cases? it's a late reply
	// TODO: why don't we update the blob? we have metahash and chunks already?
	for _, file := range reply.Responses {
		n.naming.Set(file.Name, []byte(file.Metahash))
		n.UpdateCatalog(file.Metahash, pkt.Header.Source)
		for _, chunkKey := range file.Chunks {
			if chunkKey == nil {
				continue
			}
			n.UpdateCatalog(string(chunkKey), pkt.Header.Source)
		}
	}

	if ok {
		n.Info().Msgf("notify search reply future %s", reply.RequestID)

		future <- reply
	} else {
		// do nothing, it is a arrive-late ack msg
		n.Info().Msgf("search reply(%s) packet %s has no future to complete", reply.RequestID, pkt.Header.PacketID)
	}

	return nil
}

// TODO: make the datasharing module also independent, like messager
func (n *node) SearchRequestMessageCallback(msg types.Message, pkt transport.Packet) error {
	n.Debug().Msg("start search request callback")
	req := msg.(*types.SearchRequestMessage)
	// update routing table
	n.SetRoutingEntry(req.Origin, pkt.Header.RelayedBy)

	n.searchReqsMu.Lock()
	if _, ok := n.searchReqs[req.RequestID]; ok {
		n.Debug().Msgf("search req %s already received, skip callback", req.RequestID)
		n.searchReqsMu.Unlock()
		return nil
	} else {
		n.searchReqs[req.RequestID] = struct{}{}
	}
	n.searchReqsMu.Unlock()
	// forward the req if budget permits
	newBudget := req.Budget - 1
	if newBudget > 0 {
		// TODO: shall we dont send back to lastRelayedBy and Origin?
		neis := n.getNeisExcept(pkt.Header.RelayedBy, req.Origin)
		neis, budgets := budgetAllocation(neis, newBudget)
		n.Debug().Msgf("started to forwarding search request to neighbors=%v, buds=%v", neis, budgets)
		for i := range neis {
			nei, bud := neis[i], budgets[i]
			newReq_ := types.SearchRequestMessage{
				RequestID: req.RequestID, Origin: req.Origin, Pattern: req.Pattern, Budget: bud}
			if err := n.unicastTypesMsg(nei, &newReq_); err != nil {
				err = fmt.Errorf("SearchRequestMessageCallback error: %w", err)
				n.Err(err).Send()
				return err
			}
		}
	}
	// check naming store and construct reply message
	// the naming should also be in the blob store
	matches := make([]types.FileInfo, 0, 10)
	reg := regexp.MustCompile(req.Pattern)
	n.naming.ForEach(func(key string, val []byte) bool {
		// key is file name; val is metahash
		// only when peer contains metafile with regards to the metahash shall we return the information
		if metafile := n.blob.Get(string(val)); reg.MatchString(key) && metafile != nil {
			file := types.FileInfo{Name: key, Metahash: string(val)}
			// parse metafile and fill the chunks, chunks actually contains hashKeys
			// but only if peer has value, shall we include this hashkey
			chunkHashKeys := strings.Split(string(metafile), peer.MetafileSep)
			chunks := make([][]byte, 0, len(chunkHashKeys))
			for _, chunkKey := range chunkHashKeys {
				// chunks = append(chunks, n.blob.Get(chunkKey))
				if n.blob.Get(chunkKey) != nil {
					chunks = append(chunks, []byte(chunkKey))
				} else {
					chunks = append(chunks, nil)
				}
			}
			file.Chunks = chunks
			n.Debug().Msgf("file %s matched pattern, metaHash=%s, metafile=%s, chunks=%v", key, string(val), string(metafile), len(chunks))
			// TODO: shall we fill chunks?
			matches = append(matches, file)
		}
		return true
	})
	n.Debug().Msgf("search reply matches constructed %v, will send it back", matches)
	reply := types.SearchReplyMessage{RequestID: req.RequestID, Responses: matches}
	// TODO: a lot of things could be substed
	// TODO: here it requires direct send bypassing routing table, and it could be sent back to
	//       the relayed by or original. if original, how could we bypass the routing table?
	if err := n.unicastTypesMsg(req.Origin, &reply); err != nil {
		err = fmt.Errorf("SearchRequestMessageCallback error: %w", err)
		n.Err(err).Send()
		return err
	}

	return nil
}
