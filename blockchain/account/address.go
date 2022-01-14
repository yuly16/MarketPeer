package account

import "encoding/hex"

type Address struct {
	addr [8]byte
}

func NewAddress(addr [8]byte) *Address {
	a := &Address{addr: addr}
	return a
}

func (a *Address) String() string {
	return hex.EncodeToString(a.addr[:])
}
