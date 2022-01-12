package impl

import (
	"errors"
	"time"

	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

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

// TODO: make it the method of messager
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
		n.addNeighbor(pack.Header.RelayedBy) // we can only ensure that relay is near us

		// 1. must check if the message is truly for the node
		// 	1.1 if yes, use `msgRegistry` to execute the callback associated with the message
		//  1.2 if no, update the `RelayedBy` field of the message
		if pack.Header.Destination == n.sock.GetAddress() {
			n.Trace().Str("addr", pack.Header.Destination).Msg("addr matched between peer and sender")
			if err := n.msgRegistry.ProcessPacket(pack); err != nil {
				var sendErr *SenderCallbackError
				if !errors.As(err, &sendErr) { // we only care about non-sender error
					n.Error().Err(err).Msg("error while processing the packet")
				}
			}

		} else {
			n.Warn().Str("pack.addr", pack.Header.Destination).Str("this.addr", n.sock.GetAddress()).Msg("unmatched addr, set relay to my.addr and redirect the pkt")
			pack.Header.RelayedBy = n.addr() // FIXME: not good, packet shall be immutable
			nextDest, err := n.send(pack)
			n.Err(err).Str("dest", pack.Header.Destination).Str("nextDest", nextDest).Str("msg", pack.Msg.String()).Str("pkt", pack.String()).Msg("relay packet")
		}
	}
}

func (n *node) applyDaemon() {

	for !n.isKilled() {
		value := <-n.consensusCallback
		// apply the value
		n.naming.Set(value.Filename, []byte(value.Metahash))
		n.Info().Msgf("consensus daemon: receive new consensus filename=%s, metahash=%s", value.Filename, value.Metahash)
	}
}

func (n *node) stabilize(interval time.Duration) {
	for !n.isKilled() {
		time.Sleep(interval)
		if err := n.chord.Stabilize(); err != nil {
			n.Err(err).Str("Stabilize error, ", n.conf.Socket.GetAddress())
		}
	}
}


func (n *node) fixFinger(interval time.Duration) {
	for !n.isKilled() {
		time.Sleep(interval)
		if err := n.chord.FixFinger(); err != nil {
			n.Err(err).Str("fixFinger error, ", n.conf.Socket.GetAddress())
		}
	}
}