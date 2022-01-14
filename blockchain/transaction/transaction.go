package transaction

import "go.dedis.ch/cs438/blockchain/account"

type Transaction struct {
	nonce int
	from  account.Address
	to    account.Address
	value int
	v     string
	r     string
	s     string
}
