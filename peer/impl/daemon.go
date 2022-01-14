package impl

import "time"

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
