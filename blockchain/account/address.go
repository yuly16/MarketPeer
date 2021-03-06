package account

import (
	"crypto/sha256"
	"encoding/hex"
)

type Address struct {
	Addr [8]byte
	Hex  string
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
	a := &Address{Addr: addr, Hex: hex.EncodeToString(addr[:])}
	return a
}

func (a *Address) String() string {
	return hex.EncodeToString(a.Addr[:])
}

func (a *Address) Equal(other *Address) bool {
	return a.Hex == other.Hex
}

// IsContract : contract addr first 4 bytes are 0
func (a *Address) IsContract() bool {
	for i := 0; i < 4; i++ {
		if int(a.Addr[i]) != 0 {
			return false
		}
	}
	return true
}
