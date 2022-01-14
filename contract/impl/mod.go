package impl

import (
	// "fmt"
	"strings"
    "encoding/json"
	
	"github.com/disiqueira/gotree" // lib for print tree structure in terminal
	"go.dedis.ch/cs438/contract"
)

// implements contract.ContractCode, maintained in contract account
type Contract struct {
	contract.SmartContract
	Contract_id 	 string
	Contract_name	 string
	Code_ast 	 	 Code
	Code_plain		 string
	State_tree		 *StateNode
	Proposer_addr	 string
	Proposer_account string
	Acceptor_addr	 string
	Acceptor_account string
}

// To record the execution state of the contract, we need to maintain
// the state tree of AST.
// It means that we are always tracking the progress of execution.
// Only focus on the types: Assumption, Condition, Ifclause, Action
type StateNode struct {
	Type 	  string
	Executed  bool
	Children  []*StateNode
}

func (s *StateNode) AppendChild(n *StateNode) {
	s.Children = append(s.Children, n)
}

func (s *StateNode) SetExecuted() {
	s.Executed = true
}

// Construct corresponding state tree, given the code AST
// The structure of AST is rather predictable, so we don't need to recursively traverse
func ConstructStateTree(code_ast *Code) *StateNode {
	state_root := StateNode{"code", false, []*StateNode{}}
	
	for i := 0; i < len(code_ast.Assumptions); i++ {
		assumption_state := StateNode{"assumption", false, []*StateNode{}}
		condition_state := StateNode{"condition", false, []*StateNode{}}
		assumption_state.AppendChild(&condition_state)
		state_root.AppendChild(&assumption_state)
	}

	for _, ifclause := range code_ast.IfClauses {
		if_state := StateNode{"if", false, []*StateNode{}}
		condition_state := StateNode{"condition", false, []*StateNode{}}
		if_state.AppendChild(&condition_state)
		for i := 0; i < len(ifclause.Actions); i++ {
			action_state := StateNode{"action", false, []*StateNode{}}
			if_state.AppendChild(&action_state)
		}
		state_root.AppendChild(&if_state)
	}

	return &state_root
}

// Create & initialize a new Code instance
func NewContract(contract_id string,
				 contract_name string,
				 plain_code string,
				 proposer_addr string,
				 proposer_account string,
				 acceptor_addr string,
				 acceptor_account string) contract.SmartContract {
	
	code, err := Parse(plain_code)
	if err != nil {
		panic(err)
	}

	return &Contract{
		Contract_id: contract_id,
		Contract_name: contract_name,
		Code_ast: code,
		Code_plain: plain_code,
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

// DisplayAST displays the code AST, convenient for debug
func DisplayAST(ast Code) string {
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

// Display the execution state AST, convenient for debug
func DisplayStateAST(ast Code, state_ast *StateNode) string {
	root := gotree.New("State")

	assumption_node := root.Add("Assumption")
	for i, assumption := range ast.Assumptions {
		assumption_node.Add(assumption.Condition.ToString() + " " + state_ast)
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

// Contract.String() outputs the contract in readable format
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



