package testing

import (
	"go.dedis.ch/cs438/blockchain"
	"go.dedis.ch/cs438/blockchain/messaging"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/peer/impl"
	"testing"
)

//type configTemplate struct {
//	registry registry.Registry
//	socket transport.Socket
//
//}
//
//func newConfigTemplate() *configTemplate {
//	temp := &configTemplate{
//		registry: standard.NewRegistry(),
//	}
//	return temp
//}
//
//// build a blockchain.FullNodeConf from configTemplate
//func (temp *configTemplate) build() *blockchain.FullNodeConf {
//	conf := &blockchain.FullNodeConf{}
//	conf.Addr = temp.socket.GetAddress()
//	conf.Messaging = messaging.NewMessager(impl.NewMessager(), temp.registry)
//	return conf
//
//}

func buildFullNodeConf(temp *configTemplate) *blockchain.FullNodeConf {
	conf := &blockchain.FullNodeConf{}
	conf.Addr = temp.sock.GetAddress()
	peerMessagerConf := buildPeerNodeConf(temp)
	conf.Messaging = messaging.NewRegistryMessager(conf.Addr, impl.NewMessager(*peerMessagerConf), temp.registry)
	return conf
}

// this is for compatability with messaging.Conf
func buildPeerNodeConf(template *configTemplate) *peer.Configuration {
	config := &peer.Configuration{}

	config.Socket = template.sock
	config.MessageRegistry = template.registry
	config.AntiEntropyInterval = template.AntiEntropyInterval
	config.HeartbeatInterval = template.HeartbeatInterval
	config.ContinueMongering = template.ContinueMongering
	config.AckTimeout = template.AckTimeout
	config.Storage = template.storage
	config.ChunkSize = template.chunkSize
	config.BackoffDataRequest = template.dataRequestBackoff
	config.TotalPeers = template.totalPeers
	config.PaxosThreshold = template.paxosThreshold
	config.PaxosID = template.paxosID
	config.PaxosProposerRetry = template.paxosProposerRetry
	return config
}

// construct a fullnode for testing purpose
func NewTestFullNode(t *testing.T, opts ...Option) *blockchain.FullNode {
	template := newConfigTemplate()
	for _, opt := range opts {
		opt(&template)
	}
	return blockchain.NewFullNode(buildFullNodeConf(&template))
}
