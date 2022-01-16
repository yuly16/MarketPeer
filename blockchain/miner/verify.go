package miner

import (
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/blockchain/transaction"
)

// TODO: wallet sync the account state? submitTxn
// Q: what if a transaction is stale? A: nonce field will handle it
func (m *Miner) verifyTxn(txn *transaction.SignedTransaction, worldState storage.KV) error {
	err := m.doVerifyTxn(txn, worldState)
	if err != nil {
		return fmt.Errorf("verify error: %w", err)
	}
	return nil
}

func (m *Miner) doVerifyTxn(txn *transaction.SignedTransaction, worldState storage.KV) error {
	// 1. verify signature and sender
	publicKey, err := crypto.Ecrecover(txn.Digest, txn.Signature)
	if err != nil {
		return err
	}
	okValidSignature := crypto.VerifySignature(publicKey, txn.Digest,
		txn.Signature[:len(txn.Signature)-1])
	if !okValidSignature {
		return fmt.Errorf("signature is invalid")
	}

	addr := account.NewAddressFromPublicKey(publicKey)
	if !addr.Equal(&txn.Txn.From) {
		return fmt.Errorf("txn.from=%s not consistent with pubKey derived addr=%s", &txn.Txn.From, addr)
	}

	// 2. verify nonce
	// fetch account state from world state
	value, err := worldState.Get(addr.String())
	if err != nil {
		return fmt.Errorf("addr=%s not in world state", addr.String())
	}
	accountState, ok := value.(*account.State)
	if !ok {
		return fmt.Errorf("corrputed world state, value=%v cannot be casted to account.State", value)
	}
	if txn.Txn.Nonce != int(accountState.Nonce) {
		return fmt.Errorf("txn.nonce=%d not equal to account nonce=%d", txn.Txn.Nonce, accountState.Nonce)
	}

	// 3. check if account's balance is enough to cover the transaction
	if int(accountState.Balance) < txn.Txn.Value {
		return fmt.Errorf("account balance=%d, cannot cover value=%d transfer",
			accountState.Balance, txn.Txn.Value)
	}

	return nil
}
