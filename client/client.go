package client

import (
	"go.dedis.ch/cs438/blockchain"
	"go.dedis.ch/cs438/peer/impl"
)

type Client struct {
	BlockChainFullNode blockchain.FullNode
	ChordNode          impl.Chord
}