package testing

import (
	"go.dedis.ch/cs438/client/client"
	"testing"
)

// construct a fullnode for testing purpose
func NewClient(t *testing.T, opts ...Option) *client.Client {
	template := newConfigTemplate()
	for _, opt := range opts {
		opt(&template)
	}
	fullNodeConf := buildFullNodeConf(&template)
	peerConf := buildPeerNodeConf(&template)
	Address := template.sock.GetAddress()
	cli := client.NewClient(fullNodeConf, peerConf, Address)
	return &cli
}
