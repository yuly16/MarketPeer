package account

import (
	"crypto/sha256"
	"encoding/hex"
)

type Address struct {
	addr [8]byte
	hex  string
}

func NewAddressFromPublicKey(pub []byte) *Address {
	h := sha256.New()
	_, err := h.Write(pub)
	if err != nil {
		panic(err)
	}
	hash := h.Sum(nil)
	addr := [8]byte{}
	copy(addr[:], hash[len(hash)-8:])
	return NewAddress(addr)
}

func NewAddress(addr [8]byte) *Address {
	a := &Address{addr: addr, hex: hex.EncodeToString(addr[:])}
	return a
}

func (a *Address) String() string {
	return hex.EncodeToString(a.addr[:])
}
