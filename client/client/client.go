package client

import (
	"go.dedis.ch/cs438/blockchain"
	"go.dedis.ch/cs438/chord"
	"go.dedis.ch/cs438/peer"
)

func NewClient(fullNodeConf *blockchain.FullNodeConf, peerConf *peer.Configuration,
	address string) Client {
	chordNode := chord.NewChord(fullNodeConf.PeerMessager, *peerConf)
	blockChainFullNode := blockchain.NewFullNode(fullNodeConf)
	return Client{
		BlockChainFullNode: blockChainFullNode,
		ChordNode: chordNode,
		Address: address,
	}
}


type Client struct {
	BlockChainFullNode *blockchain.FullNode
	ChordNode          *chord.Chord
	Address            string
	stat               int32
}


func (c *Client) AddPeers(address string) {
	c.ChordNode.AddPeer(address)
}

func (c *Client) Start() {
	c.ChordNode.Start()
	c.BlockChainFullNode.Start()
}

func (c *Client) Stop() {
	c.ChordNode.Stop()
	c.BlockChainFullNode.Stop()
}