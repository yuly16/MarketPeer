package miner

import (
	"fmt"
    "encoding/json"

	"go.dedis.ch/cs438/contract/impl"
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
	}

	// value transfer
	err := m.doValueTransfer(txn, worldState)
	if err != nil {
		return fmt.Errorf("execute value transfer error: %w", err)
	}
	return nil
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
	// TODO: how do we rollback the impact on the kv?
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

func RetrieveState(address string, worldState storage.KV) (*account.State, error) {
	value, err := worldState.Get(address)
	if err != nil {
		return nil, fmt.Errorf("address dont exist: %w", err)
	}
	state, ok := value.(*account.State)
	if !ok {
		return nil, fmt.Errorf("state is corrupted: %v", state)
	}
	return state, nil
}

// Contract execution logistics
func (m *Miner) doContract(txn *transaction.SignedTransaction, worldState storage.KV) error {
	// 1. reconstruct the contract code in account state
	var contract_inst impl.Contract
	contract_address := txn.Txn.To.String()
	contract_acc_state, contract_state_err := RetrieveState(contract_address, worldState)
	if contract_state_err != nil {
		return contract_state_err
	}

	contract_bytecode := contract_acc_state.CodeHash
	unmarshal_err := json.Unmarshal(contract_bytecode, &contract_inst)
	if unmarshal_err != nil {
		return fmt.Errorf("unmarshal contract byte code error: %w", unmarshal_err)
	}

	// 2. check conditions and collect actions in contract
	valid, validate_err := contract_inst.ValidateAssumptions(worldState)
	if validate_err != nil {
		return fmt.Errorf("validating contract assumptions got stuck: %w", validate_err)
	}
	if !valid {
		return nil
	}
	actions, clause_err := contract_inst.CollectActions(worldState)
	if clause_err != nil {
		return fmt.Errorf("contract if clauses fail to evaluate: %w", clause_err)
	}

	// 3. execute the transaction actions, perform on world state and submit once
	// actions: transfer, send
	proposer_account := contract_inst.GetProposerAccount()
	acceptor_account := contract_inst.GetAcceptorAccount()
	proposer_state, proposer_state_err := RetrieveState(proposer_account, worldState)
	acceptor_state, acceptor_state_err := RetrieveState(acceptor_account, worldState)

	for _, action := range actions {
		if action.Action == "transfer" { // manipulate balance
			transfer_amount := *action.Params[0].Number
			if action.Role == "buyer" {
				proposer_state.Balance -= uint(transfer_amount)
				acceptor_state.Balance += uint(transfer_amount)
			} else if action.Role == "seller" {
				proposer_state.Balance += uint(transfer_amount)
				acceptor_state.Balance -= uint(transfer_amount)
			}
		} else if action.Action == "send" { // manipulate storage
			send_product := *action.Params[0].String
			send_amount := *action.Params[1].Number
			if action.Role == "buyer" {
				amount_hold, err := proposer_state.StorageRoot.Get(send_product)
				if err != nil {
					return err
				}
				amount_hold_float, ok := amount_hold.(float64)
				if !ok {
					return fmt.Errorf("cannot cast to float: %v", amount_hold)
				}
				if amount_hold_float < send_amount {
					return fmt.Errorf("do not have enough product: %s", send_product)
				}
				proposer_state.StorageRoot.Put(send_product, amount_hold_float-send_amount)

				// manipulate acceptor state
				amount_acceptor, err := acceptor_state.StorageRoot.Get(send_product)
				if err != nil {
					acceptor_state.StorageRoot.Put(send_product, send_amount)
				} else {
					amount_acceptor_float, ok := amount_acceptor.(float64)
					if !ok {
						return fmt.Errorf("cannot cast to float: %v", amount_acceptor)
					}
					acceptor_state.StorageRoot.Put(send_product, send_amount+amount_acceptor_float)
				}
			} else if action.Role == "seller" {
				amount_hold, err := acceptor_state.StorageRoot.Get(send_product)
				if err != nil {
					return err
				}
				amount_hold_float, ok := amount_hold.(float64)
				if !ok {
					return fmt.Errorf("cannot cast to float: %v", amount_hold)
				}
				if amount_hold_float < send_amount {
					return fmt.Errorf("do not have enough product: %s", send_product)
				}
				acceptor_state.StorageRoot.Put(send_product, amount_hold_float-send_amount)

				// manipulate acceptor state
				amount_acceptor, err := proposer_state.StorageRoot.Get(send_product)
				if err != nil {
					proposer_state.StorageRoot.Put(send_product, send_amount)
				} else {
					amount_acceptor_float, ok := amount_acceptor.(float64)
					if !ok {
						return fmt.Errorf("cannot cast to float: %v", amount_acceptor)
					}
					proposer_state.StorageRoot.Put(send_product, send_amount+amount_acceptor_float)
				}
			}
		}
	}
	if err_put_proposer := worldState.Put(proposer_account, proposer_state); err_put_proposer != nil {
		return fmt.Errorf("cannot put proposer addr and state to KV: %w", err_put_proposer)
	}
	if err_put_acceptor := worldState.Put(acceptor_account, acceptor_state); err_put_acceptor != nil {
		return fmt.Errorf("cannot put acceptor addr and state to KV: %w", err_put_acceptor)
	}
	proposer_state.Nonce += 1

	return nil
}
