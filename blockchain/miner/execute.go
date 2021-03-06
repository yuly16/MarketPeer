package miner

import (
	"encoding/json"
	"fmt"

	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/blockchain/transaction"
	"go.dedis.ch/cs438/contract/impl"
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
	if txn.Txn.To.IsContract() && txn.Txn.Type != transaction.CREATE_CONTRACT {
		return m.doContract(txn, worldState)
	} else if txn.Txn.Type == transaction.CREATE_CONTRACT {
		// create a transaction
		err := m.createContract(txn, worldState)
		if err != nil {
			return fmt.Errorf("create contract error: %w", err)
		}
		return nil
	} else if txn.Txn.Type == transaction.SET_STORAGE {
		if err := m.setStorage(txn, worldState); err != nil {
			return fmt.Errorf("set storage error: %w", err)
		}
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

func (m *Miner) setStorage(txn *transaction.SignedTransaction, worldState storage.KV) error {
	value, err := worldState.Get(txn.Txn.From.String())
	if err != nil {
		return fmt.Errorf("from address dont exist: %w", err)
	}
	fromState, ok := value.(*account.State)
	if !ok {
		return fmt.Errorf("from state is corrupted: %v", fromState)
	}
	fromState.Nonce += 1
	if err := fromState.StorageRoot.Put(txn.Txn.StoreK, txn.Txn.StoreV); err != nil {
		return err
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
	value, err = worldState.Get(txn.Txn.To.String())
	if err != nil {
		return fmt.Errorf("to address dont exist: %w", err)
	}
	toState, ok := value.(*account.State)
	if !ok {
		return fmt.Errorf("to state is corrupted: %v", fromState)
	}

	fromState.Balance -= uint(txn.Txn.Value)
	fromState.Nonce += 1
	if err = worldState.Put(txn.Txn.From.String(), fromState); err != nil {
		return fmt.Errorf("cannot put from addr and state to KV: %w", err)
	}

	// 2. increase receiver balance
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

func NumericToFloat(num interface{}) (float64, bool) {
	if num_float, ok := num.(float64); ok {
		return num_float, true
	} else if num_int, ok := num.(int); ok {
		return float64(num_int), true
	} else {
		return 0, false
	}
}

// Contract execution logistics
// Type casting problem: need send(string, int)
func (m *Miner) doContract(txn *transaction.SignedTransaction, worldState storage.KV) error {
	// 1. reconstruct the contract code in account state
	var contract_inst impl.Contract
	contract_address := txn.Txn.To.String()
	contract_acc_state, contract_state_err := RetrieveState(contract_address, worldState)
	if contract_state_err != nil {
		return contract_state_err
	}

	contract_bytecode := []byte(contract_acc_state.Code)
	unmarshal_err := json.Unmarshal(contract_bytecode, &contract_inst)
	if unmarshal_err != nil {
		return fmt.Errorf("unmarshal contract byte code error: %w", unmarshal_err)
	}

	// we need to first check whether or not triggered by acceptor account
	// otherwise, the transaction will not be executed
	if txn.Txn.From.String() != contract_inst.GetAcceptorAccount() {
		return fmt.Errorf("contract not triggered by acceptor: %s", txn.Txn.From.String())
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
	if proposer_state_err != nil {
		return fmt.Errorf("fail to retrieve proposer state: %w", proposer_state_err)
	}
	acceptor_state, acceptor_state_err := RetrieveState(acceptor_account, worldState)
	if acceptor_state_err != nil {
		return fmt.Errorf("fail to retrieve acceptor state: %w", acceptor_state_err)
	}

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
				amount_hold_float, ok := NumericToFloat(amount_hold)
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
					amount_acceptor_float, ok := NumericToFloat(amount_acceptor)
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
				amount_hold_float, ok := NumericToFloat(amount_hold)
				if !ok {
					return fmt.Errorf("cannot cast to float: %T", amount_hold)
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
					amount_acceptor_float, ok := NumericToFloat(amount_acceptor)
					if !ok {
						return fmt.Errorf("cannot cast to float: %v", amount_acceptor)
					}
					proposer_state.StorageRoot.Put(send_product, send_amount+amount_acceptor_float)
				}
			}
		}
	}

	acceptor_state.Nonce += 1 // increment nonce
	if err_put_proposer := worldState.Put(proposer_account, proposer_state); err_put_proposer != nil {
		return fmt.Errorf("cannot put proposer addr and state to KV: %w", err_put_proposer)
	}
	if err_put_acceptor := worldState.Put(acceptor_account, acceptor_state); err_put_acceptor != nil {
		return fmt.Errorf("cannot put acceptor addr and state to KV: %w", err_put_acceptor)
	}

	return nil
}

func (m *Miner) createContract(txn *transaction.SignedTransaction, worldState storage.KV) error {
	state := account.State{
		Nonce:       0,
		Balance:     0,
		StorageRoot: m.kvFactory(),
		Code:        txn.Txn.Code,
	}
	from_state, err := RetrieveState(txn.Txn.From.String(), worldState)
	if err != nil {
		return fmt.Errorf("Fail to retrive target contract: ", err)
	}
	from_state.Nonce += 1
	if err := worldState.Put(txn.Txn.From.String(), from_state); err != nil {
		return fmt.Errorf("cannot put from addr and state to KV: %w", err)
	}
	err = worldState.Put(txn.Txn.To.String(), &state)
	if err != nil {
		return fmt.Errorf("put contract error: %w", err)
	}
	return nil
}
