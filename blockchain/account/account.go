package account

import (
	"fmt"
	"go.dedis.ch/cs438/blockchain/storage"
)

// Account in the Epfer network
// https://ethereum.org/en/developers/docs/accounts/
type Account struct {
	addr  *Address
	state *State
}

func (a *Account) GetAddr() *Address {
	return a.addr
}

func (a *Account) GetState() *State {
	return a.state
}

func (a *Account) String() string {
	return fmt.Sprintf("{addr: %s, state: %s}", a.addr, a.state)
}

type AccountBuilder struct {
	addr  *Address
	state *StateBuilder
}

func NewAccountBuilder(pub []byte, kvFactory storage.KVFactory) *AccountBuilder {
	return &AccountBuilder{addr: NewAddressFromPublicKey(pub), state: NewStateBuilder(kvFactory)}
}

func (ab *AccountBuilder) WithBalance(balance uint) *AccountBuilder {
	ab.state.SetBalance(balance)
	return ab
}

func (ab *AccountBuilder) WithKV(key string, value interface{}) *AccountBuilder {
	ab.state.SetKV(key, value)
	return ab
}

func (ab *AccountBuilder) Build() *Account {
	return &Account{ab.addr, ab.state.Build()}
}
