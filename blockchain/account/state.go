package account

import (
	"fmt"

	"go.dedis.ch/cs438/blockchain/storage"
)

// State field are exported since it needs to be marshalled
type State struct {
	Nonce       uint       // number of transactions created
	Balance     uint       // number of Epfer/Fei owned
	StorageRoot storage.KV // storage state, it is a KV
	CodeHash    []byte     // codeHash, only for contract account. empty for external account
}

type StateBuilder struct {
	balance     uint
	codeHash	[]byte
	storageRoot storage.KV
}

func NewStateBuilder(kvFactory storage.KVFactory) *StateBuilder {
	return &StateBuilder{storageRoot: kvFactory()}
}

func (sb *StateBuilder) SetBalance(balance uint) *StateBuilder {
	sb.balance = balance
	return sb
}

func (sb *StateBuilder) SetKV(key string, value interface{}) *StateBuilder {
	if err := sb.storageRoot.Put(key, value); err != nil {
		panic(err)
	}
	return sb
}

func (sb *StateBuilder) SetCode(bytecode []byte) *StateBuilder {
	sb.codeHash = bytecode
	return sb
}
func (sb *StateBuilder) Build() *State {
	s := State{
		Nonce:       0,
		Balance:     sb.balance,
		StorageRoot: sb.storageRoot,
		CodeHash: 	 sb.codeHash,
	}
	return &s
}

func NewState(kvFactory storage.KVFactory) *State {
	s := State{
		Nonce:    0,
		Balance:  0,
		CodeHash: []byte{},
	}
	s.StorageRoot = kvFactory()
	return &s
}

func (s *State) StorageHash() string {
	return s.StorageRoot.Hash()
}

func (s *State) String() string {
	return fmt.Sprintf("{nonce=%d, balance=%d, storageHash=%s, codeHash=%s}",
		s.Nonce, s.Balance, s.StorageHash()[:8]+"...", s.CodeHash)
}
