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
	Code        string     // code
}

// FIXME: can we return *State while also implementing Copyable?
func (s *State) Copy() storage.Copyable {
	ret := &State{}
	ret.Nonce = s.Nonce
	ret.Balance = s.Balance
	ret.Code = s.Code
	ret.StorageRoot = s.StorageRoot.Copy()
	return ret
}

type StateBuilder struct {
	nonce       uint
	balance     uint
	storageRoot storage.KV
	code        string
}

func NewStateBuilder(kvFactory storage.KVFactory) *StateBuilder {
	return &StateBuilder{storageRoot: kvFactory()}
}

func (sb *StateBuilder) SetCode(code string) *StateBuilder {
	sb.code = code

	return sb
}

func (sb *StateBuilder) SetNonce(nonce uint) *StateBuilder {
	sb.nonce = nonce
	return sb
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

func (sb *StateBuilder) Build() *State {
	s := State{
		Nonce:       sb.nonce,
		Balance:     sb.balance,
		StorageRoot: sb.storageRoot,
		Code:        sb.code,
	}
	return &s
}

func NewState(kvFactory storage.KVFactory) *State {
	s := State{
		Nonce:   0,
		Balance: 0,
	}
	s.StorageRoot = kvFactory()
	return &s
}

func (s *State) StorageHash() string {
	return s.StorageRoot.Hash()
}

func (s *State) String() string {
	return fmt.Sprintf("{nonce=%d, balance=%d, storageHash=%s, code=%s}",
		s.Nonce, s.Balance, s.StorageHash()[:8]+"...", s.Code)
}
