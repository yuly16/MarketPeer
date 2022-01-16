package miner

import (
	"fmt"

	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/blockchain/transaction"
)

func (m *Miner) executeTxn(txn *transaction.SignedTransaction, worldState storage.KV) error {
	err := m.doExecuteTxn(txn, worldState)
	if err != nil {
		return fmt.Errorf("execute error: %w", err)
	}
	return nil
}

func (m *Miner) doExecuteTxn(txn *transaction.SignedTransaction, worldState storage.KV) error {
	// TODO: contract case
	if txn.Txn.To.IsContract() {
		return m.doContract(txn, worldState)
	} else if txn.Txn.Type == transaction.CREATE_CONTRACT {
		// create a transaction
		return nil
	} else {
		// value transfer
		err := m.doValueTransfer(txn, worldState)
		if err != nil {
			return fmt.Errorf("execute value transfer error: %w", err)
		}
		return nil
	}

}

func (m *Miner) doValueTransfer(txn *transaction.SignedTransaction, worldState storage.KV) error {
	// 1. deduct sender balance, increase sender nonce
	value, err := worldState.Get(txn.Txn.From.String())
	if err != nil {
		return fmt.Errorf("from address dont exist: %w", err)
	}
	fromState, ok := value.(*account.State)
	if !ok {
		return fmt.Errorf("from state is corrupted: %v", fromState)
	}
	fromState.Balance -= uint(txn.Txn.Value)
	fromState.Nonce += 1
	if err = worldState.Put(txn.Txn.From.String(), fromState); err != nil {
		return fmt.Errorf("cannot put from addr and state to KV: %w", err)
	}

	// 2. increase receiver balance
	value, err = worldState.Get(txn.Txn.To.String())
	if err != nil {
		return fmt.Errorf("to address dont exist: %w", err)
	}
	toState, ok := value.(*account.State)
	if !ok {
		return fmt.Errorf("to state is corrupted: %v", fromState)
	}
	toState.Balance += uint(txn.Txn.Value)
	if err = worldState.Put(txn.Txn.To.String(), toState); err != nil {
		return fmt.Errorf("cannot put from addr and state to KV: %w", err)
	}

	return nil
}

func (m *Miner) doContract(txn *transaction.SignedTransaction, worldState storage.KV) error {

	return nil
}
