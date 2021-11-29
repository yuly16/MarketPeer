package impl

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
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

type OnConsensusReach func(types.PaxosValue) error

type Consensus interface {
	// from the perspective of consensus layer, the value could be everything
	Propose(value types.PaxosValue) (<-chan types.PaxosValue, error)
	Stop() error
}

type Clock interface {
	StepAndMaxID() (uint, uint)
}

func NewConsensus(on OnConsensusReach, messager peer.Messaging, callback chan *types.PaxosValue, conf peer.Configuration) Consensus {
	mpaxos := &MultiPaxos{Messaging: messager}
	mpaxos.Logger = _logger.With().Str("Paxos", fmt.Sprintf("%d %s", conf.PaxosID, conf.Socket.GetAddress())).Logger()
	mpaxos.msgRegistry = conf.MessageRegistry
	mpaxos.conf = conf
	mpaxos.paxosID_ = conf.PaxosID
	mpaxos.proposalID = conf.PaxosID
	mpaxos.promiseCh = make(map[uint]chan *types.PaxosPromiseMessage)
	mpaxos.promiseCh[0] = make(chan *types.PaxosPromiseMessage, conf.TotalPeers)
	mpaxos.acceptCh = make(chan *types.PaxosAcceptMessage, conf.TotalPeers)
	mpaxos.tlcCh = make(chan *types.TLCMessage, conf.TotalPeers)

	mpaxos.addr = conf.Socket.GetAddress()
	mpaxos.applyCallback = callback
	mpaxos.livestat = ALIVE
	mpaxos.blockchain = conf.Storage.GetBlockchainStore()
	mpaxos.learnerAdvance = make(chan struct{}) // synchronization, no buffer
	mpaxos.learnerSignalPropose = make(map[uint]chan types.PaxosValue)
	mpaxos.stepFinish = make(map[uint]chan types.PaxosValue) // it might also be step-indexed
	mpaxos.stepTLCMsgs = make(map[uint][]*types.TLCMessage)
	mpaxos.learnerSignalPropose[0] = make(chan types.PaxosValue, 1)
	mpaxos.stepFinish[0] = make(chan types.PaxosValue, 1) // synchronization, no buffer
	mpaxos.msgRegistry.RegisterMessageCallback(types.PaxosPrepareMessage{}, mpaxos.PaxosPrepareCallback)
	mpaxos.msgRegistry.RegisterMessageCallback(types.PaxosProposeMessage{}, mpaxos.PaxosProposeCallback)
	mpaxos.msgRegistry.RegisterMessageCallback(types.PaxosAcceptMessage{}, mpaxos.PaxosAcceptCallback)
	mpaxos.msgRegistry.RegisterMessageCallback(types.PaxosPromiseMessage{}, mpaxos.PaxosPromiseCallback)
	mpaxos.msgRegistry.RegisterMessageCallback(types.TLCMessage{}, mpaxos.TLCCallback)

	mpaxos.onConsensusReach = on
	go mpaxos.tlcDaemon()
	go mpaxos.learnerRoutine(0, mpaxos.learnerSignalPropose[0])
	return mpaxos
}

// implementes Clock
type ThresholdLogicalClock struct {
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

	applyCallback chan *types.PaxosValue

	msgRegistry registry.Registry
	conf        peer.Configuration
	paxosID_    uint // or proposal ID
	addr        string

	blockchain storage.Store

	promiseCh map[uint]chan *types.PaxosPromiseMessage
	acceptCh  chan *types.PaxosAcceptMessage
	tlcCh     chan *types.TLCMessage

	stepTLCMsgs map[uint][]*types.TLCMessage // only store step>=my.step

	mu                   sync.Mutex
	step                 uint
	proposalID           uint // perspective of proposer
	tlcSent              bool // whether tlc is already sent in current step
	learnerAdvance       chan struct{}
	learnerSignalPropose map[uint]chan types.PaxosValue
	stepFinish           map[uint]chan types.PaxosValue
	onConsensusReach     OnConsensusReach

	proposalMaxID uint // perspective of acceptor

	phase    int32
	livestat int32

	acceptedValue *types.PaxosValue
	acceptedID    uint
}

func (mp *MultiPaxos) Stop() error {
	atomic.StoreInt32(&mp.livestat, KILL)
	return nil
}

// FIXME: can we abstract a TLC layer?
// assumption: step = tlc.Step
func (mp *MultiPaxos) onTLCReachedAtCurrentStep(tlc *types.TLCMessage, catchingup bool, __logger zerolog.Logger) error {
	__logger.Info().Msgf("onTLCReachedAtCurrentStep: catchingup=%v, tlc=%s", catchingup, *tlc)
	// reach consensus at current step!
	mp.mu.Lock()
	if mp.step != tlc.Step {
		__logger.Info().Msgf("state changed, tlcmsg.step=%d != mystep=%d, value=%s, catchingup=%v", tlc.Step, mp.step, tlc.Block.Value, catchingup)
		mp.mu.Unlock()

		return nil
	}
	mp.mu.Unlock()
	__logger.Info().Msgf("collect enough tlc message step=%d, value=%v, catchingup=%v", tlc.Step, tlc.Block.Value, catchingup)
	// 1. add the block to block chain
	mp.mu.Lock()
	// check if state has been changed
	if mp.step != tlc.Step {
		mp.mu.Unlock()
		return nil
	}
	block := tlc.Block
	hexhash := hex.EncodeToString(block.Hash)
	// blockBytes = make([]byte, block.)
	blockBytes, err := block.Marshal()
	if err != nil {
		mp.mu.Unlock()
		return err
	}
	mp.blockchain.Set(hexhash, blockBytes)
	mp.blockchain.Set(storage.LastBlockKey, block.Hash)

	// 2. set the namestore
	// mp.applyCallback <- &tlc.Block.Value // TODO: might cause deadlock, pay attention
	mp.onConsensusReach(tlc.Block.Value)
	// 3. broadcast if
	//    (1) we haven't broadcast for current step
	//    (2) not catching up
	broadcast := !catchingup
	if !catchingup {
		broadcast = broadcast && !mp.tlcSent
		if broadcast { // only when we are sure we would broadcast, shall we update tlcSent
			mp.tlcSent = true // prevent sent twice in critical section
		}
	}
	mp.mu.Unlock()

	if broadcast {
		tlc_, err := mp.msgRegistry.MarshalMessage(tlc)
		if err != nil {
			err = fmt.Errorf("tlcDaemon error: %w, exit", err)
			__logger.Err(err).Send()
			panic(err) // fatal error, we just panic
		}
		__logger.Info().Msgf("will broadcast tlc=%s", tlc)
		mp.Messaging.Broadcast(tlc_)
		__logger.Info().Msgf("broadcasted tlc=%s", tlc)
	}
	// 4. advance step 这里一定要注意, 涉及到状态改变的临界区域了
	mp.mu.Lock()
	if mp.step == tlc.Step {
		mp.advanceClockAsStep(tlc.Step, tlc.Block.Value)
	} else {
		__logger.Info().Msgf("state changed, tlcmsg.step=%d != mystep=%d", tlc.Step, mp.step)
		mp.mu.Unlock()
		return nil
	}
	mp.mu.Unlock()

	// see if we could catch up
	if msgs, ok := mp.stepTLCMsgs[tlc.Step+1]; ok && len(msgs) >= mp.conf.PaxosThreshold(mp.conf.TotalPeers) {
		mp.onTLCReachedAtCurrentStep(msgs[0], true, __logger)
	}

	return nil
}

func (mp *MultiPaxos) isKilled() bool {
	return atomic.LoadInt32(&mp.livestat) == KILL
}

// TODO: actually this daemon dont need to check the step, since only this could modify the step
func (mp *MultiPaxos) tlcDaemon() {
	__logger := mp.Logger.With().Str("daemon", "tlcDaemon").Logger()
	__logger.Info().Msg("start")
	// collect the tlc messages and advance the step accordingly
	for !mp.isKilled() {
		// event-based, check for current-step consensus

		tlc := <-mp.tlcCh
		__logger.Info().Msgf("receive tlc=%s", tlc)

		mp.mu.Lock()
		if tlc.Step < mp.step {
			// ignore outdated tlc
			__logger.Info().Msgf("tlc.step=%d < mp.step=%d, continue", tlc.Step, mp.step)

			mp.mu.Unlock()
			continue
		}
		mp.mu.Unlock()
		msgs, ok := mp.stepTLCMsgs[tlc.Step]
		if !ok {
			mp.stepTLCMsgs[tlc.Step] = []*types.TLCMessage{tlc}
			msgs = mp.stepTLCMsgs[tlc.Step]
		} else {
			// update stepTLCMsgs
			msgs = append(msgs, tlc)
			mp.stepTLCMsgs[tlc.Step] = msgs
		}

		if len(msgs) < mp.conf.PaxosThreshold(mp.conf.TotalPeers) {
			continue
		}

		// it needs to be at the same current step
		mp.mu.Lock()
		if mp.step != tlc.Step {
			__logger.Info().Msgf("though tlc collected enough for step=%d, we are at step=%d", tlc.Step, mp.step)
			mp.mu.Unlock()
			continue
		}
		mp.mu.Unlock()

		mp.onTLCReachedAtCurrentStep(tlc, false, __logger)
	}
}

func (mp *MultiPaxos) onConsensusReachAtStep(step uint, paxosValue types.PaxosValue) error {
	__logger := mp.Logger.With().Str("routine", "learner").Uint("step", step).Logger()

	// construct the block for blockchain
	prevHash := mp.blockchain.Get(storage.LastBlockKey)
	if prevHash == nil {
		prevHash = make([]byte, 32)
	}
	beforeHash := []byte(strconv.Itoa(int(step)) + paxosValue.UniqID + paxosValue.Filename + paxosValue.Metahash + string(prevHash))
	hashkey, err := sha256(beforeHash)
	if err != nil {
		return err
	}
	block := types.BlockchainBlock{
		Index:    step,
		Value:    paxosValue,
		PrevHash: prevHash,
		Hash:     hashkey,
	}

	__logger.Info().Msgf("assemble a new block {block n°%d H(%x) - %v - %x", block.Index, block.Hash[:int(math.Min(4, float64(len(block.Hash))))], block.Value, block.PrevHash[:int(math.Min(4, float64(len(block.PrevHash))))])
	// assemble a new TLC message
	tlc_ := types.TLCMessage{
		Step:  step,
		Block: block,
	}
	// FIXME: sick of following pattern, over and over again, messaging interface is shit
	tlc, err := mp.msgRegistry.MarshalMessage(tlc_)
	if err != nil {
		return err
	}
	// we did not check the step and tlcSent here, since it might cause deadlock
	// (we are waiting for lock, advanceCLock waiting for learnerAdvance)
	// otherwise, we check it in the TLCCallback, which will be called in broadcast
	// so here, we directly broadcast and inform the proposer
	__logger.Info().Msgf("assemble a tlc: %s, will broadcast it", tlc_.String())

	err = mp.Messaging.Broadcast(tlc)
	if err != nil {
		return err
	}
	__logger.Info().Msgf("broadcasted tlc: %s", tlc_.String())

	return nil
}

// every step, we renew a learnerRoutine?
// we need to make sure that idAccepts only contains current step's value
// reinit at AdvanceClock is not enough, it might receive a step=0, then step++
// but it will update the idAccepts who is after-reinit
// if we restart another learnerRoutine, then what if the old one receive a message on the acceptCh?
// 所以我们要保证 step 一旦更新, 上一个 learnerRoutine 必须立即停止
func (mp *MultiPaxos) learnerRoutine(step uint, signalPropose chan<- types.PaxosValue) {
	__logger := mp.Logger.With().Str("routine", "learner").Uint("step", step).Logger()
	__logger.Info().Msg("start")
	// Q: why groupby the uniqID of PaxosValue, rather than accept.ID?
	// A: to increase liveness, since in Paxos larger ID will propose same value, they dont need
	//    to contend with each other.
	// idAccepts for step
	idAccepts := make(map[string][]*types.PaxosAcceptMessage)
	for {
		select {
		case accept := <-mp.acceptCh:
			__logger.Info().Msgf("learner receives accept=%s", accept)
			if accept.Step != step {
				panic(fmt.Sprintf("accept.Step=%d != learner.step=%d", accept.Step, step)) // Note: it is a sanity check
			}
			paxosValue := accept.Value
			if msgs, ok := idAccepts[paxosValue.UniqID]; !ok {
				idAccepts[paxosValue.UniqID] = []*types.PaxosAcceptMessage{accept}
			} else {
				idAccepts[paxosValue.UniqID] = append(msgs, accept)
			}
			// complexity comes from that, when we are in this select branch, step might change
			if len(idAccepts[paxosValue.UniqID]) >= mp.conf.PaxosThreshold(mp.conf.TotalPeers) {
				// consensus reached on this uniqID
				// if we are proposer, then we need to signal the proposer routine, such that proposer wont do phase1 again

				// it also might cause deadlock.... channel usage is not so good here, we should use a conditional variable?
				// TODO: here we could add a note, we could use atomic variable to solve this scenario(fine-grained lock also works)
				// we cannot lock before the <-mp.learnerAdvance
				// now another problem on the synchronization between propose and learner
				// Q: what if we send a learnerSignalPropose, but propose already timeout and start another round
				// A: does not matter, this signal will be catched up at next loop of propose
				// Q: old step problem? will old-step propose consume the new signal?
				// learner sends signal; step changed; propose receive signal; does not matter, it is a stale signal anyway
				// step changed; learner not know yet, sends signal; propose receive signal; does not matter, signal is still stale
				// step changed, learner new routine, sends signal; propose old not yet exit(timer is very long) -> problem!
				// solution: (1) step-indexed signaling (2) if step changed, then old propose should exit instantly?
				__logger.Info().Msgf("consensus reach at value=%s", paxosValue)
				if atomic.LoadInt32(&mp.phase) == PHASETWO {

					signalPropose <- paxosValue
					__logger.Info().Msgf("notify propose that we have reached consensus", paxosValue)
				}
				err := mp.onConsensusReachAtStep(step, paxosValue)
				__logger.Err(err).Send()

				// if we are acceptor, then we could just broadcast the TLC
				<-mp.learnerAdvance // simply wait for the advance command
				return              // we cannot directly return, otherwise advanceClock would stuck
			}
		case <-mp.learnerAdvance:
			return
		}
	}

}

// TODO: stepFinish need to synchronized between Tag and us, is this right?
func (mp *MultiPaxos) Propose(value types.PaxosValue) (<-chan types.PaxosValue, error) {
	var err error = nil
	mp.mu.Lock()
	phase := atomic.LoadInt32(&mp.phase)
	step := mp.step
	mp.Info().Msgf("try to Propose %v at step %d", value, step)
	defer func() {
		mp.mu.Unlock()
		if err == nil { // could propose
			mp.propose(value, step)
		}
	}()

	if phase == NOTPROPOSE {
		mp.transitPhaseAsStep(step, PHASEONE)
		mp.Info().Msgf("could propose at step=%d, transit to phase1", step)

	} else {
		mp.Info().Msgf("we are proposing at step=%d, phase=%d, return err=OnProposing", step, phase)

		err = ErrOnProposing
	}

	// TODO: finish mp.stepFinish semantics
	return mp.stepFinish[step], err
}

// not thread-safe, rely on callers
func (mp *MultiPaxos) transitPhaseAsStep(step uint, newPhase int32) {
	// curPhase := atomic.LoadInt32(&mp.phase) // TODO: could also do some phase sanity checking
	if step == mp.step {
		atomic.StoreInt32(&mp.phase, newPhase)
	}
}

// Note: rely on caller to make it thread-safe
func (mp *MultiPaxos) advanceClockAsStep(step uint, paxosValue types.PaxosValue) {
	if step != mp.step {
		return
	}
	mp.step += 1
	mp.proposalID = mp.paxosID_ // reinit
	mp.proposalMaxID = 0
	mp.acceptedValue = nil
	mp.acceptedID = 0
	mp.tlcSent = false
	mp.learnerAdvance <- struct{}{} // make sure last learner routine exits, then we do our learnerRoutine
	mp.acceptCh = make(chan *types.PaxosAcceptMessage, mp.conf.TotalPeers)
	// flush all the values in leanrerSignalPropose, since we need a new start
	mp.learnerSignalPropose[mp.step] = make(chan types.PaxosValue, 1)
	delete(mp.learnerSignalPropose, mp.step-1)
	delete(mp.stepTLCMsgs, mp.step-1) // delete old step
	go mp.learnerRoutine(mp.step, mp.learnerSignalPropose[mp.step])
	mp.stepFinish[mp.step] = make(chan types.PaxosValue, 1)
	mp.transitPhaseAsStep(mp.step, NOTPROPOSE) // before finish, then next Propose would know we are not proposing
	mp.promiseCh[mp.step] = make(chan *types.PaxosPromiseMessage, mp.conf.TotalPeers)
	delete(mp.promiseCh, mp.step-1)
	mp.stepFinish[mp.step-1] <- paxosValue // this would make sure the signal is sent,
	// TODO: fuck the test, they did not call the Tag, so I cant directly test this synchrnization...
	// delete(mp.stepFinish, mp.step-1)       // delete old step

	mp.Info().Msgf("advance step to %d, init proposalID to %d, init phase to %d", mp.step, mp.proposalID, mp.phase)
}

// assumption: we are in the step, if it violates, exit immediately without side effects
// phase changing is linear just in propose routine. We need to take care of step
// Note: next propose is guaranteed to be of next step. It is an important property
func (mp *MultiPaxos) propose(value types.PaxosValue, step uint) error {
	__logger := mp.Logger.With().Str("func", "propose").Uint("step", step).Logger()
	defer func() {
		mp.mu.Lock()
		mp.transitPhaseAsStep(step, NOTPROPOSE)
		__logger.Info().Msgf("enter NOTPROPOSE state")

		mp.mu.Unlock()
	}()

	mp.mu.Lock()
	// we make sure we are in the NOPROPOSE state at this line
	// After enter phase1, we could tho also receive promise from last phase loop, but it's fine with Paxos
	promiseCh := mp.promiseCh[step] // clear
	mp.mu.Unlock()

	// phase 1
	// 1. broadcast PaxosPrepareMessage

	highestAccpetID := uint(0)
	proposedValue := value
	for {
		// check step at start of iteration
		mp.mu.Lock()
		mp.transitPhaseAsStep(step, PHASEONE)
		__logger.Info().Msg("enter phase 1")

		if step != mp.step {
			mp.mu.Unlock()
			return nil
		}
		prepare_ := types.PaxosPrepareMessage{
			Step:   step,
			ID:     mp.proposalID,
			Source: mp.addr,
		}
		mp.mu.Unlock()

		__logger.Info().Msgf("assemble a new prepare msg %s, will broadcast it", prepare_)
		prepare, err := mp.msgRegistry.MarshalMessage(&prepare_)
		if err != nil {
			return err
		}
		// dont need to check step here, since callback would check it anyway
		// and callback would throw err, and here we would directly return
		err = mp.Messaging.Broadcast(prepare)
		if err != nil {
			return err
		}
		__logger.Info().Msgf("broadcasted prepare msg %s", prepare_)

		// wait for the of threshold of peers
		timer := time.After(mp.conf.PaxosProposerRetry)
		promised := true
		for i := 0; i < mp.conf.PaxosThreshold(mp.conf.TotalPeers); i++ {
			select {
			case promise := <-promiseCh:
				if promise.AcceptedID > highestAccpetID {
					// accept value are those accepted by the majority, so we need
					// to follow this consensus. If there is accepted value, it is
					// guaranteed to exist in any majority set, so it is guranteed to
					// be found by us. Then we just do an "echo"
					// Q: when do we set our accepeted value and accepted term? broadcast?
					// A: when we propose, we will update our accepted value
					highestAccpetID = promise.AcceptedID
					proposedValue = *promise.AcceptedValue
				}

			case <-timer:
				mp.mu.Lock()
				// step has been changed, dont let it affect our proposalID state
				if mp.step != step {
					mp.mu.Unlock()
					return nil
				}
				mp.proposalID += mp.conf.TotalPeers
				__logger.Info().Msgf("phase 1 timeout, increment proposalID to %d", mp.proposalID)
				mp.mu.Unlock()
				promised = false

			}

		}
		if !promised {
			continue
		}
		__logger.Info().Msgf("receive %d promise messages", mp.conf.PaxosThreshold(mp.conf.TotalPeers))

		break // we have received all promise message
	}

	// after phase 1, majority of the nodes promise that they will not accept proposal
	// with ID smaller than this.proposalID

	// phase 2, after we collected the promise
	mp.mu.Lock()
	// check step before we propose
	if mp.step != step {
		mp.mu.Unlock()
		return nil
	}
	mp.transitPhaseAsStep(step, PHASETWO)
	__logger.Info().Msgf("enter phase 2, proposeID=%d, proposeValue=%s", mp.proposalID, proposedValue)
	propose_ := types.PaxosProposeMessage{
		Step:  step,
		ID:    mp.proposalID,
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
	__logger.Info().Msgf("assemble a new propose msg %s, and will broadcast it", propose_)

	// dont need to check step here, same reason as prepare broadcast
	err = mp.Messaging.Broadcast(propose)
	if err != nil {
		return err
	}
	__logger.Info().Msgf("propose msg %s broadcasted", propose_)

	// wait for the of threshold of peers
	timer := make(chan struct{}) // could signal all
	go func() {
		<-time.After(mp.conf.PaxosProposerRetry)
		close(timer)
	}()
	mp.mu.Lock()
	signal, ok := mp.learnerSignalPropose[step]
	if !ok { // step has already be changed, we only care about valid step's signal
		mp.mu.Unlock()
		return nil
	}
	mp.mu.Unlock()

	select {
	case <-signal:
		// consensus reached
		__logger.Info().Msgf("proposer knows consensus reached")

	case <-timer:
		// another phase1-2 loop
		mp.mu.Lock()
		// check step, since it might affect proposalID
		if mp.step != step {
			mp.mu.Unlock()
			return nil
		}
		mp.proposalID += mp.conf.TotalPeers
		__logger.Info().Msgf("phase 2 timeout, increment proposalID to %d", mp.proposalID)
		mp.mu.Unlock()

		// start another phase, TODO: it might cause stack overflow
		return mp.propose(value, step)
	}

	return nil
}

func (mp *MultiPaxos) PaxosPrepareCallback(msg types.Message, pkt transport.Packet) error {
	__logger := mp.Logger.With().Str("func", "PaxosPrepareCallback").Logger()
	prepare := msg.(*types.PaxosPrepareMessage)
	__logger.Info().Msgf("enter PaxosPrepareCallback, msg=%s", prepare.String())
	mp.mu.Lock()
	if prepare.Step != mp.step {
		__logger.Info().Msgf("tlc.step=%d != msg.step=%d, return", mp.step, prepare.Step)
		err := fmt.Errorf("state has changed: tlc.step=%d != msg.step=%d", mp.step, prepare.Step)
		mp.mu.Unlock()
		return err
	}
	if prepare.ID <= mp.proposalMaxID {
		__logger.Info().Msgf("tlc.maxID=%s >= msg.id=%s, return", mp.proposalMaxID, prepare.ID)
		err := fmt.Errorf("state has changed: tlc.maxID=%d >= msg.id=%d", mp.proposalMaxID, prepare.ID)
		mp.mu.Unlock()
		return err

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
	mp.mu.Unlock()

	promise, err := mp.msgRegistry.MarshalMessage(&promise_)
	if err != nil {
		__logger.Err(err).Send()
		return err
	}

	// it is evoked by Broadcast self process packet, we dont need to send a private msg
	// here, just directly send on promiseCh
	// if prepare.Source == mp.addr {
	// 	mp.Messaging.Unicast(mp.addr, promise)
	// 	return nil
	// }

	// now it is truly acceptor receiving the prepare messages, we then need to
	// assemble a Private msg and send it back
	private_ := types.PrivateMessage{
		Recipients: map[string]struct{}{prepare.Source: {}},
		Msg:        &promise,
	}
	private, err := mp.msgRegistry.MarshalMessage(&private_)

	if err != nil {
		__logger.Err(err).Send()
		return err
	}

	__logger.Info().Msgf("will broadcast private=%s", private)
	err = mp.Messaging.Broadcast(private)
	__logger.Info().Msgf("broadcasted private=%s", private)

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
		err := fmt.Errorf("state has changed! tlc.step=%d != msg.step=%d", mp.step, propose.Step)
		mp.mu.Unlock()
		return err
	}
	if propose.ID != mp.proposalMaxID {
		__logger.Info().Msgf("tlc.maxID=%s != msg.id=%s, return", mp.proposalMaxID, propose.ID)
		err := fmt.Errorf("state has changed! tlc.maxID=%d != msg.id=%d", mp.proposalMaxID, propose.ID)
		mp.mu.Unlock()
		return err
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
	__logger.Info().Msgf("accept=%s broadcasted", accept)

	return nil
}

func (mp *MultiPaxos) PaxosPromiseCallback(msg types.Message, pkt transport.Packet) error {
	__logger := mp.Logger.With().Str("func", "PaxosPromiseCallback").Logger()
	promise := msg.(*types.PaxosPromiseMessage)
	__logger.Info().Msgf("enter PaxosPromiseCallback, msg=%s", promise.String())
	mp.mu.Lock()
	// Note: the returned error is intended for the sender routine to do state check before broadcast
	//       for receiver, dont bother printing these redundant msgs
	if promise.Step != mp.step {
		__logger.Info().Msgf("tlc.step=%s != msg.step=%s, return", mp.step, promise.Step)
		err := fmt.Errorf("state has changed. tlc.step=%d != msg.step=%d", mp.step, promise.Step)
		mp.mu.Unlock()
		return err
	}
	phase := atomic.LoadInt32(&mp.phase)
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
	mp.promiseCh[promise.Step] <- promise // send the promise
	__logger.Info().Msgf("send the promise the promiseCh, promise=%s", *promise)

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
		err := fmt.Errorf("state has changed. tlc.step=%d != msg.step=%d", mp.step, accept.Step)
		mp.mu.Unlock()
		return err
	}
	// TODO: do we really need to use an atomic value, or even lock?
	phase := atomic.LoadInt32(&mp.phase)
	// Q: what if a peer is accepting as well trying to propose
	// A:  1. if we first enter the phase2, then receive the accept, then we should process it
	//     2. if accept before phase2, then it has to be accept regarding to propose broadcasted by others
	//        so, we still do nothing about it
	//     3. possibility of concurrently happening? it will not happen for accept regarding our sent
	//        propose. what if we are just entering the phase2, but not broadcasted yet, then we receive
	//        an accept from others?
	// if phase == NOTPROPOSE {
	// 	// we are just acceptor, so dont need to process this message anyway
	// 	__logger.Info().Msgf("acceptor accept callback, do nothing, return nil")
	// 	mp.mu.Unlock()
	// 	return nil
	// }
	// NOTPROPOSE=not a proposer, so if it is a proposer, then it need to be in the PHASETWO
	if phase != NOTPROPOSE && phase != PHASETWO {
		__logger.Warn().Msgf("proposer not in phase 2, but %d, but we still proceed", phase)
		// mp.mu.Unlock()
		// return fmt.Errorf("proposer not in phase 2 but %d", phase)
		// return nil
	}
	// now we are in phase2 and same step
	// we then could send the accept
	select {
	case mp.acceptCh <- accept:
		__logger.Info().Msgf("send accept=%s to acceptCh", *accept)
	default:
		__logger.Warn().Msgf("acceptch is blocked, skip send accept=%s", *accept)
	}

	mp.mu.Unlock()
	return nil
}

func (mp *MultiPaxos) TLCCallback(msg types.Message, pkt transport.Packet) error {
	__logger := mp.Logger.With().Str("func", "TLCCallback").Logger()
	tlc := msg.(*types.TLCMessage)
	__logger.Info().Msgf("enter TLCCallback, msg=%s, pkt=%s", tlc.String(), pkt.String())
	mp.mu.Lock()
	defer mp.mu.Unlock()
	// defer mp.mu.Unlock()
	// if accept.Step != mp.step {
	// 	__logger.Info().Msgf("tlc.step=%s != msg.step=%s, return", mp.step, accept.Step)
	// 	mp.mu.Unlock()
	// 	return fmt.Errorf("state has changed. tlc.step=%d != msg.step=%d", mp.step, accept.Step)
	// }
	if mp.step > tlc.Step {
		return &SenderCallbackError{fmt.Errorf("stale tlc of step=%d < mp.step=%d", tlc.Step, mp.step)}
	}
	if mp.tlcSent {
		mp.tlcCh <- tlc
		return &SenderCallbackError{fmt.Errorf("TLC is already broadcasted at step %d", mp.step)}
	} else {
		mp.tlcSent = true // set it to true to prevent other TLC broadcast
	}
	mp.tlcCh <- tlc
	__logger.Info().Msgf("send to tlcCh, tlc=%s", tlc)
	return nil
}
