package tests

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"

	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/storage"
)

var accountFactory = func(balance uint, key string, value interface{}) (*account.Account, *ecdsa.PrivateKey) {
	privateKey1, _ := crypto.GenerateKey()
	publicKey1 := &privateKey1.PublicKey
	ac1 := account.NewAccountBuilder(crypto.FromECDSAPub(publicKey1), storage.CreateSimpleKV).
		WithBalance(balance).WithKV(key, value).Build()
	return ac1, privateKey1
}