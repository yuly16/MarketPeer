package wallet

import "fmt"

// Account is application-level
type AccountInfo struct {
	Addr    string
	Balance int
	Storage map[string]interface{}
}

func (a AccountInfo) String() string {
	return fmt.Sprintf("{Addr=%s, Balance=%d, Storage=%v}", a.Addr, a.Balance, a.Storage)
}