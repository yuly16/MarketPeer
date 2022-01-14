package account

import (
	"fmt"
	"go.dedis.ch/cs438/blockchain/storage"
)

// State field are exported since it needs to be marshalled
type State struct {
	// TODO: 4 fields
	Nonce       uint // number of transactions created
	Balance     uint // number of Epfer/Fei owned
	StorageHash string
	StorageRoot storage.KV // storage state, it is a KV
	CodeHash    string     // codeHash, only for contract account. empty for external account
}

type StateBuilder struct {
	balance     uint
	storageRoot storage.KV
}

func NewStateBuilder(kvFactory storage.KVFactory) *StateBuilder {
	return &StateBuilder{storageRoot: kvFactory()}
}

func (sb *StateBuilder) SetBalance(balance uint) *StateBuilder {
	sb.balance = balance
	return sb
}

func (sb *StateBuilder) Build() *State {
	s := State{
		Nonce:       0,
		Balance:     sb.balance,
		StorageRoot: sb.storageRoot,
		StorageHash: sb.storageRoot.Hash(),
	}
	return &s
}

func NewState(kvFactory storage.KVFactory) *State {
	s := State{
		Nonce:    0,
		Balance:  0,
		CodeHash: "",
	}
	s.StorageRoot = kvFactory()
	s.StorageHash = s.StorageRoot.Hash()
	return &s
}

func (s *State) String() string {
	return fmt.Sprintf("{nonce=%d, balance=%d, storageHash=%s, codeHash=%s}",
		s.Nonce, s.Balance, s.StorageHash, s.CodeHash)
}
