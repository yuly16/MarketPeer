package account

type Address struct {
	addr [20]byte
}

func (a *Address) String() string {
	return string(a.addr[:])
}
