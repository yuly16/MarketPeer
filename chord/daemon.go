package chord

import (
	"fmt"
	"time"
)

const (
	KILL = iota
	ALIVE
)

func (c *Chord) stabilizeDaemon(interval time.Duration) {
	for !c.isKilled() {
		time.Sleep(interval)
		if err := c.Stabilize(); err != nil {
			c.Err(err).Str("Stabilize error, ", c.conf.Socket.GetAddress())
		}
	}
	fmt.Println("stabilizeDaemon: I am stop!")
}


func (c *Chord) fixFingerDaemon(interval time.Duration) {
	for !c.isKilled() {
		time.Sleep(interval)
		if err := c.FixFinger(); err != nil {
			c.Err(err).Str("fixFinger error, ", c.conf.Socket.GetAddress())
		}
	}
	fmt.Println("fixFingerDaemon: I am stop!")
}

