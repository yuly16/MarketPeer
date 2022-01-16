package tests

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
	"go.dedis.ch/cs438/blockchain/block"
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

var contractFactory = func(balance uint, bytecode []byte) (*account.Account, *ecdsa.PrivateKey) {
	privateKey1, _ := crypto.GenerateKey()
	// publicKey1 := &privateKey1.PublicKey
	ac1 := account.NewContractBuilder([8]byte{0, 0, 0, 0, 0, 0, 0, 1}, 
		storage.CreateSimpleKV).WithBalance(balance).WithCode(string(bytecode)).Build()
	return ac1, privateKey1
}

// based on init accounts info, generate the genesis block
func generateGenesisBlock(kvFactory storage.KVFactory, accounts ...*account.Account) *block.Block {
	bb := block.NewBlockBuilder(kvFactory).
		SetParentHash(block.DUMMY_PARENT_HASH).
		SetNonce(0).
		SetNumber(0).
		SetDifficulty(2).
		SetBeneficiary(*account.NewAddress([8]byte{}))
	for _, acc := range accounts {
		bb.SetAddrState(acc.GetAddr(), acc.GetState())
	}
	b := bb.Build()
	return b
}