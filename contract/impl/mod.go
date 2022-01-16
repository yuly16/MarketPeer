package impl

import (
	// "fmt"
	"reflect"
	"strings"
    "encoding/json"
	
	"golang.org/x/xerrors"
	"github.com/disiqueira/gotree" // lib for print tree structure in terminal
	"go.dedis.ch/cs438/contract"
	"go.dedis.ch/cs438/contract/parser"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/storage"
)

// implements contract.ContractCode, maintained in contract account
type Contract struct {
	contract.SmartContract
	Contract_id 	 string
	Contract_name	 string
	Code_ast 	 	 parser.Code
	Code_plain		 string
	State_tree		 *StateNode
	Proposer_addr	 string
	Proposer_account string
	Acceptor_addr	 string
	Acceptor_account string
}

// Create & initialize a new Code instance
func NewContract(contract_id string,
				 contract_name string,
				 plain_code string,
				 proposer_addr string,
				 proposer_account string,
				 acceptor_addr string,
				 acceptor_account string) contract.SmartContract {
	
	code, err := parser.Parse(plain_code)
	if err != nil {
		panic(err)
	}
	state_ast := ConstructStateTree(&code)

	return &Contract{
		Contract_id: contract_id,
		Contract_name: contract_name,
		Code_ast: code,
		Code_plain: plain_code,
		State_tree: state_ast,
		Proposer_addr: proposer_addr,
		Proposer_account: proposer_account,
		Acceptor_addr: acceptor_addr,
		Acceptor_account: acceptor_account,
	}
}

// Marshal marshals the Contract instance into a byte representation.
// Marshal and Unmarshal to transfer the contract instance in packet
func (c *Contract) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

// Unmarshal unmarshals the data into the current Contract instance.
func Unmarshal(data []byte, contract *Contract) error {
	return json.Unmarshal(data, contract)
}

func (c *Contract) GetCodeAST() parser.Code {
	return c.Code_ast
}

func (c *Contract) GetStateAST() *StateNode {
	return c.State_tree
}

func (c *Contract) GetProposerAccount() string {
	return c.Proposer_account
}

func (c *Contract) GetAcceptorAccount() string {
	return c.Acceptor_account
}

// Check the condition from the world state of the underlying blockchain
func (c *Contract) ValidateCondition(condition parser.Condition, worldState storage.KV) (bool, error) {
	role := condition.Object.Role
	fields := condition.Object.Fields
	op := condition.Op
	val := condition.Value

	// evaluate and retrieve the compared value
	var target_account string
	if role == "buyer" {
		target_account = c.Proposer_account
	} else if role == "seller" {
		target_account = c.Acceptor_account
	}

	// we assume the fields restricted to balance / storage key
	var left_val interface{}
	if len(fields) > 1 {
		return false, xerrors.Errorf("Condition field unknown, assumed only one attribute.")
	}
	attribute := fields[0].Name

	// retrieve value corresponding to role.fields from the world state
	value, err := worldState.Get(target_account)
	if err != nil {
		panic(err)
	}
	account_state, ok := value.(*account.State)
	if !ok {
		return false, xerrors.Errorf("account state is corrupted: %v", account_state)
	}
	if attribute == "balance" {
		left_val = float64(account_state.Balance)
	} else {
		left_val, err = account_state.StorageRoot.Get(attribute)
		if reflect.TypeOf(left_val).String() == "int" {
			left_val = float64(left_val.(int))
		}
		if err != nil {
			return false, xerrors.Errorf("key not exist in storage: %v", attribute)
		}
	}

	var right_val interface{}
	if val.String == nil {
		right_val = *val.Number
	} else {
		right_val = *val.String
	}

	// type checking
	if reflect.TypeOf(left_val) != reflect.TypeOf(right_val) {
		return false, xerrors.Errorf("type of the values are not paired: %v %v", left_val, right_val)
	}
	// type assertion
	if reflect.TypeOf(left_val).String() == "string" {
		var left_string = left_val.(string)
		var right_string = right_val.(string)
		return CompareString(left_string, right_string, op)
	} else if reflect.TypeOf(left_val).String() == "float64" {
		var left_number = left_val.(float64)
		var right_number = right_val.(float64)
		return CompareNumber(left_number, right_number, op)
	} else {
		return false, xerrors.Errorf("type not supported: %v", reflect.TypeOf(left_val))
	}
}

func CompareString(left_val string, right_val string, op string) (bool, error) {
	switch op {
	case "==":
		return (left_val == right_val), nil
	case "!=":
		return (left_val != right_val), nil
	}
	return false, xerrors.Errorf("comparator not supported on string: %v", op)
}

func CompareNumber(left_val float64, right_val float64, op string) (bool, error) {
	switch op {
	case ">":
		return (left_val > right_val), nil 
	case "<":
		return (left_val < right_val), nil 							
	case ">=":
		return (left_val >= right_val), nil
	case "<=":
		return (left_val <= right_val), nil
	case "==":
		return (left_val == right_val), nil
	case "!=":
		return (left_val != right_val), nil
	}
	return false, xerrors.Errorf("comparator not supported on number: %v", op)
}

// Need to firstly check the validity of assumptions
func (c *Contract) ValidateAssumptions(worldState storage.KV) (bool, error) {
	assumptions_valid := true
	for i, assumption := range c.Code_ast.Assumptions {
		condition := assumption.Condition
		condition_valid, err := c.ValidateCondition(condition, worldState)
		if err != nil {
			return false, err
		}
		if condition_valid == false {
			assumptions_valid = false
		} else { // synchronize the validity to the state tree
			c.State_tree.Children[i].SetValid()
			c.State_tree.Children[i].Children[0].SetValid()
		}
		
		c.State_tree.Children[i].SetExecuted()
		c.State_tree.Children[i].Children[0].SetExecuted()
	}

	return assumptions_valid, nil
}

// Evaluate the If clauses and collect the actions (with true condition)
// The result includes all the actions satisfied their conditions
func (c *Contract) CollectActions(worldState storage.KV) ([]parser.Action, error) {
	var execute_actions []parser.Action
	number_of_assumptions := len(c.Code_ast.Assumptions)

	// loop for all if clauses in the AST
	for i, ifclause := range c.Code_ast.IfClauses {
		condition := ifclause.Condition
		condition_valid, err := c.ValidateCondition(condition, worldState)
		if err != nil {
			return []parser.Action{}, err	
		}
		ifclause_state := c.State_tree.Children[i + number_of_assumptions]
		condition_state := ifclause_state.Children[0]

		if condition_valid == false {
			ifclause_state.SetExecuted()
			condition_state.SetExecuted()
		} else {
			ifclause_state.SetExecuted()
			ifclause_state.SetValid()
			condition_state.SetExecuted()
			condition_state.SetValid()

			for j := 1; j < len(ifclause_state.Children); j++ {
				ifclause_state.Children[j].SetExecuted()
			}
			for _, action := range ifclause.Actions {
				execute_actions = append(execute_actions, *action)
			}
		}
	}

	return execute_actions, nil
}

// DisplayAST displays the code AST, convenient for debug
func DisplayAST(ast parser.Code) string {
	root := gotree.New("Code")

	assumption_node := root.Add("Assumption")
	for _, assumption := range ast.Assumptions {
		assumption_node.Add(assumption.Condition.ToString())
	}

	for _, ifclause := range ast.IfClauses {
		if_node := root.Add("If Clause")
		if_node.Add(ifclause.Condition.ToString())
		for _, action := range ifclause.Actions {
			if_node.Add(action.ToString())
		}
	}

	AST_out := root.Print()
	return AST_out
}

// Contract.String() outputs the contract in pretty readable format
func (c Contract) String() string {
	out := new(strings.Builder)

	out.WriteString("=================================================================\n")
	out.WriteString("| Contract: " + c.Contract_name + "\n")
	out.WriteString("| ID: " + c.Contract_id + "\n")
	out.WriteString("=================================================================\n")
	out.WriteString("| Proposer: [" + c.Proposer_addr + "] (" + c.Proposer_account + ")\n")
	out.WriteString("| Acceptor: [" + c.Acceptor_addr + "] (" + c.Acceptor_account + ")\n")
	out.WriteString("=================================================================\n")
	out.WriteString("| Contract code: " + "\n")
	out.WriteString(c.Code_plain + "\n")
	out.WriteString("=================================================================\n")

	return out.String()
}

