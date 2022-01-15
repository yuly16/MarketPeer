package client

import (
	"go.dedis.ch/cs438/blockchain"
	"go.dedis.ch/cs438/chord"
)

type Client struct {
	BlockChainFullNode blockchain.FullNode
	ChordNode          chord.Chord
}