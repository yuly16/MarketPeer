package account

import (
	"fmt"
	"go.dedis.ch/cs438/blockchain/storage"
)

type State struct {
	// TODO: 4 fields
	nonce       uint32     // number of transactions created
	balance     uint32     // number of Epfer owned
	storageRoot storage.KV // storage state, it is a KV
	codeHash    string     // codeHash, only for contract account. empty for external account
}

func NewState(kvFactory storage.KVFactory) *State {
	s := State{
		nonce:    0,
		balance:  0,
		codeHash: "",
	}
	s.storageRoot = kvFactory()
	return &s
}

func (s *State) String() string {
	return fmt.Sprintf("{nonce=%d, balance=%d, storageRoot=%s, codeHash=%s}",
		s.nonce, s.balance, s.storageRoot.Hash(), s.codeHash)
}
