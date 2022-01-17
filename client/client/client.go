package client

import (
	"encoding/json"
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

func (c *Client) StoreProduct(key uint, product Product) error {
	err := c.ChordNode.Put(key, product)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) StoreProductString(key string, product Product) error {
	err := c.StoreProduct(c.ChordNode.HashKey(key), product)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) ReadProduct(key uint) (Product, bool) {
	value, ok, _ := c.ChordNode.Get(key)
	valueBytes, _ := json.Marshal(value)
	product := Product{}
	if ok {
		_ = json.Unmarshal(valueBytes, &product)
	}
	return product, ok
}

func (c *Client) ReadProductString(key string) (Product, bool) {
	return c.ReadProduct(c.ChordNode.HashKey(key))
}


func (c *Client) StoreAccount(key string, account Account) error {
	err := c.ChordNode.Put(c.ChordNode.HashKey(key), account)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) ReadAccount(key string) (Account, bool) {
	value, ok, _ := c.ChordNode.Get(c.ChordNode.HashKey(key))
	valueBytes, _ := json.Marshal(value)
	account := Account{}
	if ok {
		_ = json.Unmarshal(valueBytes, &account)
	}
	return account, ok
}