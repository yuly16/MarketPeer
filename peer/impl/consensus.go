package impl

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/registry"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

type Consensus interface {
	// from the perspective of consensus layer, the value could be everything
	Propose(value types.PaxosValue) error
}

type Clock interface {
	StepAndMaxID() (uint, uint)
}

func NewConsensus(messager peer.Messaging, conf peer.Configuration) Consensus {
	mpaxos := &MultiPaxos{Messaging: messager}
	mpaxos.Logger = _logger.With().Str("Paxos", fmt.Sprintf("%d %s", conf.PaxosID, conf.Socket.GetAddress())).Logger()
	mpaxos.msgRegistry = conf.MessageRegistry
	mpaxos.conf = conf
	mpaxos.paxosID = conf.PaxosID
	mpaxos.promiseCh = make(chan *types.PaxosPromiseMessage, conf.TotalPeers)
	mpaxos.acceptCh = make(chan *types.PaxosAcceptMessage, conf.TotalPeers)
	mpaxos.addr = conf.Socket.GetAddress()

	mpaxos.msgRegistry.RegisterMessageCallback(types.PaxosPrepareMessage{}, mpaxos.PaxosPrepareCallback)
	mpaxos.msgRegistry.RegisterMessageCallback(types.PaxosProposeMessage{}, mpaxos.PaxosProposeCallback)
	mpaxos.msgRegistry.RegisterMessageCallback(types.PaxosAcceptMessage{}, mpaxos.PaxosAcceptCallback)
	mpaxos.msgRegistry.RegisterMessageCallback(types.PaxosPromiseMessage{}, mpaxos.PaxosPromiseCallback)

	return mpaxos
}

// implementes Clock
type ThresholdLogicalClock struct {
	// TODO: do we need lock to protect step?
	step  uint
	maxID uint
}

func (tlc *ThresholdLogicalClock) StepAndMaxID() (uint, uint) {
	return tlc.step, tlc.maxID
}

// paxos proposer phase
// phase transition:
// 0 -> 1
// 1 -> 2
// 2 -> 0
// 2 -> 1
const (
	NOTPROPOSE = iota // not propose
	PHASEONE          // at phase 1
	PHASETWO          // at phase 2
)

// implements Consensus
// depends on socket, msgRegistry
type MultiPaxos struct {
	peer.Messaging
	zerolog.Logger

	msgRegistry registry.Registry
	conf        peer.Configuration
	paxosID     uint // or proposal ID
	addr        string

	promiseCh chan *types.PaxosPromiseMessage
	acceptCh  chan *types.PaxosAcceptMessage

	mu            sync.Mutex
	step          uint
	proposalMaxID uint

	phase int32

	acceptedValue *types.PaxosValue
	acceptedID    uint
}

func (mp *MultiPaxos) Propose(value types.PaxosValue) error {
	// ret := make(chan struct{}) // TODO: shall we make it buffered?
	// go func() {
	// 	mp.propose(value)
	// 	ret <- struct{}{}
	// }()

	return mp.propose(value)
}

func (mp *MultiPaxos) propose(value types.PaxosValue) error {
	mp.mu.Lock()
	// we make sure we are in the NOPROPOSE state at this line
	// After enter phase1, we could tho also receive promise from last phase loop, but it's fine with Paxos
	mp.promiseCh = make(chan *types.PaxosPromiseMessage, mp.conf.TotalPeers) // clear
	mp.acceptCh = make(chan *types.PaxosAcceptMessage, mp.conf.TotalPeers)   // clear
	mp.mu.Unlock()

	// phase 1
	// 1. broadcast PaxosPrepareMessage
	atomic.StoreInt32(&mp.phase, PHASEONE)
	mp.Info().Msg("enter phase 1")
	highestAccpetID := uint(0)
	proposedValue := value
	for {
		mp.mu.Lock()
		prepare_ := types.PaxosPrepareMessage{
			Step:   0, // TODO
			ID:     mp.paxosID,
			Source: mp.addr, // TODO: abstraction leak
		}
		mp.mu.Unlock()

		mp.Info().Msgf("assemble a new prepare msg %s", prepare_)
		prepare, err := mp.msgRegistry.MarshalMessage(&prepare_)
		if err != nil {
			return err
		}
		err = mp.Messaging.Broadcast(prepare)
		if err != nil {
			return err
		}

		// wait for the of threshold of peers
		timer := time.After(mp.conf.PaxosProposerRetry)
		for i := 0; i < mp.conf.PaxosThreshold(mp.conf.TotalPeers); i++ {
			select {
			case promise := <-mp.promiseCh:
				if promise.AcceptedID > highestAccpetID {
					// accept value are those accepted by the majority, so we need
					// to follow this consensus. If there is accepted value, it is
					// guaranteed to exist in any majority set, so it is guranteed to
					// be found by us. Then we just do an "echo"
					// TODO: when do we set our accepeted value and accepted term? broadcast?
					highestAccpetID = promise.AcceptedID
					proposedValue = *promise.AcceptedValue
				}

			case <-timer:
				mp.mu.Lock()
				mp.paxosID += mp.conf.TotalPeers
				mp.Info().Msgf("phase 1 timeout, increment paxosID to %d", mp.paxosID)
				mp.mu.Unlock()
				continue
			}

		}
		mp.Info().Msgf("receive %d promise messages", mp.conf.PaxosThreshold(mp.conf.TotalPeers))

		break // we have received all promise message
	}

	// after phase 1, majority of the nodes promise that they will not accept proposal
	// with ID smaller than this.paxosID

	// phase 2, after we collected the promise
	mp.mu.Lock()
	atomic.StoreInt32(&mp.phase, PHASETWO) // acceptCh should be inited before phase changing
	mp.Info().Msgf("enter phase 2, proposeID=%d, proposeValue=%s", mp.paxosID, proposedValue)
	propose_ := types.PaxosProposeMessage{
		Step:  0, // TODO
		ID:    mp.paxosID,
		Value: proposedValue,
	}
	mp.mu.Unlock()

	// assume that we are in the same Step, so we should at least ensure that
	// when we send the packet to other peers, we are in the same state?
	// but if the callback will lock themselves, so Broadcast cannot be in the locked region
	// if the state has changed and it will be found by self process-packet, then we should not
	// send the packet.
	propose, err := mp.msgRegistry.MarshalMessage(propose_)
	if err != nil {
		return err
	}
	err = mp.Messaging.Broadcast(propose)
	if err != nil {
		return err
	}

	// wait for the of threshold of peers
	timer := make(chan struct{}) // could signal all
	go func() {
		<-time.After(mp.conf.PaxosProposerRetry)
		close(timer)
	}()
	signal := make(chan types.PaxosValue, 1)
	// Q: why groupby the uniqID of PaxosValue, rather than accept.ID?
	// A: to increase liveness, since in Paxos larger ID will propose same value, they dont need
	//    to contend with each other.
	idAccepts := make(map[string][]*types.PaxosAcceptMessage)
	go func() {
		for {
			select {
			case accept := <-mp.acceptCh:
				paxosValue := accept.Value
				if msgs, ok := idAccepts[paxosValue.UniqID]; !ok {
					idAccepts[paxosValue.UniqID] = []*types.PaxosAcceptMessage{accept}
				} else {
					idAccepts[paxosValue.UniqID] = append(msgs, accept)
				}
				if len(idAccepts[paxosValue.UniqID]) >= mp.conf.PaxosThreshold(mp.conf.TotalPeers) {
					// consensus reached on this uniqID
					signal <- paxosValue
					return
				}
			case <-timer:
				return // dont stuck forever
			}
		}

	}()
	select {
	case <-signal:
		// consensus reached
		mp.Info().Msgf("consensus reached on value=%s", mp.conf.PaxosThreshold(mp.conf.TotalPeers))

	case <-timer:
		// another phase1-2 loop
		mp.mu.Lock()
		mp.paxosID += mp.conf.TotalPeers
		mp.Info().Msgf("phase 2 timeout, increment paxosID to %d", mp.paxosID)
		mp.mu.Unlock()

		// start another phase, TODO: it might cause stack overflow
		atomic.StoreInt32(&mp.phase, NOTPROPOSE)
		return mp.propose(value)
	}

	// TODO: examine it, will phase be changed in other routine?
	atomic.StoreInt32(&mp.phase, NOTPROPOSE)
	mp.Info().Msgf("enter NOTPROPOSE state")

	return nil
}

func (mp *MultiPaxos) PaxosPrepareCallback(msg types.Message, pkt transport.Packet) error {
	__logger := mp.Logger.With().Str("func", "PaxosPrepareCallback").Logger()
	prepare := msg.(*types.PaxosPrepareMessage)
	__logger.Info().Msgf("enter PaxosPrepareCallback, msg=%s", prepare.String())
	mp.mu.Lock()
	if prepare.Step != mp.step {
		__logger.Info().Msgf("tlc.step=%d != msg.step=%d, return", mp.step, prepare.Step)
		mp.mu.Unlock()
		return fmt.Errorf("state has changed: tlc.step=%d != msg.step=%d", mp.step, prepare.Step)
	}
	if prepare.ID <= mp.proposalMaxID {
		__logger.Info().Msgf("tlc.maxID=%s <= msg.id=%s, return", mp.proposalMaxID, prepare.ID)
		mp.mu.Unlock()
		return fmt.Errorf("state has changed: tlc.maxID=%d <= msg.id=%d", mp.proposalMaxID, prepare.ID)

	}

	// then we issue a PromiseMessage
	// we should directly advance the maxID, the paper says
	// return "less than n that it has **accepted**", not maxID it received
	mp.proposalMaxID = prepare.ID
	promise_ := types.PaxosPromiseMessage{
		Step:          mp.step,
		ID:            mp.proposalMaxID,
		AcceptedID:    mp.acceptedID,
		AcceptedValue: mp.acceptedValue,
	}
	promise, err := mp.msgRegistry.MarshalMessage(&promise_)
	if err != nil {
		__logger.Err(err).Send()
		mp.mu.Unlock()
		return err
	}

	private_ := types.PrivateMessage{
		Recipients: map[string]struct{}{prepare.Source: {}},
		Msg:        &promise,
	}
	private, err := mp.msgRegistry.MarshalMessage(&private_)

	mp.mu.Unlock()
	if err != nil {
		__logger.Err(err).Send()
		return err
	}

	__logger.Info().Msgf("will broadcast private=%s", private)
	err = mp.Messaging.Broadcast(private)
	if err != nil {
		__logger.Err(err).Send()
		return err
	}

	return nil
}

func (mp *MultiPaxos) PaxosProposeCallback(msg types.Message, pkt transport.Packet) error {
	__logger := mp.Logger.With().Str("func", "PaxosProposeCallback").Logger()
	propose := msg.(*types.PaxosProposeMessage)
	__logger.Info().Msgf("enter PaxosProposeCallback, msg=%s", propose.String())
	mp.mu.Lock()
	if propose.Step != mp.step {
		__logger.Info().Msgf("tlc.step=%s != msg.step=%s, return", mp.step, propose.Step)
		mp.mu.Unlock()
		return fmt.Errorf("state has changed! tlc.step=%d != msg.step=%d", mp.step, propose.Step)
	}
	if propose.ID != mp.proposalMaxID {
		__logger.Info().Msgf("tlc.maxID=%s != msg.id=%s, return", mp.proposalMaxID, propose.ID)
		mp.mu.Unlock()
		return fmt.Errorf("state has changed! tlc.maxID=%d != msg.id=%d", mp.proposalMaxID, propose.ID)
	}
	mp.acceptedID = propose.ID
	mp.acceptedValue = &propose.Value
	accept_ := types.PaxosAcceptMessage{
		Step:  mp.step,
		ID:    mp.proposalMaxID,
		Value: propose.Value,
	}
	accept, err := mp.msgRegistry.MarshalMessage(&accept_)

	mp.mu.Unlock()
	if err != nil {
		__logger.Err(err).Send()
		return err
	}
	__logger.Info().Msgf("will broadcast accept=%s", accept)
	err = mp.Messaging.Broadcast(accept)
	if err != nil {
		__logger.Err(err).Send()
		return err
	}
	return nil
}

func (mp *MultiPaxos) PaxosPromiseCallback(msg types.Message, pkt transport.Packet) error {
	// TODO: promise callback do nothing? what about accept value?
	__logger := mp.Logger.With().Str("func", "PaxosPromiseCallback").Logger()
	promise := msg.(*types.PaxosPromiseMessage)
	__logger.Info().Msgf("enter PaxosPromiseCallback, msg=%s", promise.String())
	mp.mu.Lock()
	// TODO: add note, the returned error is intended for the sender routine to do state check before broadcast
	//       for receiver, dont bother printing these redundant msgs
	if promise.Step != mp.step {
		__logger.Info().Msgf("tlc.step=%s != msg.step=%s, return", mp.step, promise.Step)
		mp.mu.Unlock()
		return fmt.Errorf("state has changed. tlc.step=%d != msg.step=%d", mp.step, promise.Step)
	}
	phase := atomic.LoadInt32(&mp.phase) // TODO: phase 这里用 atomic 而没有在 lock 里面有风险
	if phase == NOTPROPOSE {
		// we are just acceptor, so dont need to process this message anyway
		__logger.Info().Msgf("acceptor promise callback, do nothing, return nil")
		mp.mu.Unlock()
		return nil
	}
	if phase != PHASEONE {
		__logger.Info().Msgf("not in phase one but %d, return", phase)
		mp.mu.Unlock()
		// return fmt.Errorf("proposer not in phase one but %d", phase)
		return nil
	}
	// now we are in the proposer.phase1
	mp.promiseCh <- promise // send the promise
	mp.mu.Unlock()
	return nil
}

func (mp *MultiPaxos) PaxosAcceptCallback(msg types.Message, pkt transport.Packet) error {
	__logger := mp.Logger.With().Str("func", "PaxosAcceptCallback").Logger()
	accept := msg.(*types.PaxosAcceptMessage)
	__logger.Info().Msgf("enter PaxosAcceptCallback, msg=%s", accept.String())
	mp.mu.Lock()
	if accept.Step != mp.step {
		__logger.Info().Msgf("tlc.step=%s != msg.step=%s, return", mp.step, accept.Step)
		mp.mu.Unlock()
		return fmt.Errorf("state has changed. tlc.step=%d != msg.step=%d", mp.step, accept.Step)
	}
	// TODO: do we really need to use an atomic value, or even lock?
	phase := atomic.LoadInt32(&mp.phase)
	// TODO: Q: what if a peer is accepting as well trying to propose
	// A:  1. if we first enter the phase2, then receive the accept, then we should process it
	//     2. if accept before phase2, then it has to be accept regarding to propose broadcasted by others
	//        so, we still do nothing about it
	//     3. possibility of concurrently happening? it will not happen for accept regarding our sent
	//        propose. what if we are just entering the phase2, but not broadcasted yet, then we receive
	//        an accept from others?
	if phase == NOTPROPOSE {
		// we are just acceptor, so dont need to process this message anyway
		__logger.Info().Msgf("acceptor accept callback, do nothing, return nil")
		mp.mu.Unlock()
		return nil
	}
	if phase != PHASETWO {
		__logger.Info().Msgf("we are not in phase 2, but %d, return", phase)
		mp.mu.Unlock()
		// return fmt.Errorf("proposer not in phase 2 but %d", phase)
		return nil
	}
	// now we are in phase2 and same step
	// we then could send the accept
	mp.acceptCh <- accept
	mp.mu.Unlock()
	return nil
}
