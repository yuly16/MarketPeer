package account

type Address struct {
	addr [20]byte
}

func NewAddress(addr [20]byte) *Address {
	a := &Address{addr: addr}
	return a
}

func (a *Address) String() string {
	return string(a.addr[:])
}
