package account

import (
	"fmt"
	"sync"

	"go.dedis.ch/cs438/blockchain/storage"
)

// Account in the Epfer network
// https://ethereum.org/en/developers/docs/accounts/
type Account struct {
	mu    sync.Mutex
	addr  *Address
	state *State
}

func (a *Account) GetAddr() *Address {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.addr
}

func (a *Account) GetNonce() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return int(a.state.Nonce)
}

func (a *Account) GetBalance() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return int(a.state.Balance)
}

func (a *Account) GetState() *State {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.state
}

func (a *Account) SetState(state *State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state = state
}

func (a *Account) String() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return fmt.Sprintf("{addr: %s, state: %s}", a.addr, a.state)
}

type AccountBuilder struct {
	addr  *Address
	state *StateBuilder
}

func NewAccountBuilder(pub []byte, kvFactory storage.KVFactory) *AccountBuilder {
	return &AccountBuilder{addr: NewAddressFromPublicKey(pub), state: NewStateBuilder(kvFactory)}
}

// TODO: for test
func NewContractBuilder(address [8]byte, kvFactory storage.KVFactory) *AccountBuilder {
	return &AccountBuilder{addr: NewAddress(address), state: NewStateBuilder(kvFactory)}
}

func (ab *AccountBuilder) WithBalance(balance uint) *AccountBuilder {
	ab.state.SetBalance(balance)
	return ab
}

func (ab *AccountBuilder) WithKV(key string, value interface{}) *AccountBuilder {
	ab.state.SetKV(key, value)
	return ab
}

func (ab *AccountBuilder) WithCode(code string) *AccountBuilder {
	ab.state.SetCode(code)
	return ab
}

func (ab *AccountBuilder) Build() *Account {
	return &Account{addr: ab.addr, state: ab.state.Build()}
}
