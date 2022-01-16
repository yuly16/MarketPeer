package impl

import (
	"strconv"
	"github.com/disiqueira/gotree" // lib for print tree structure in terminal
	"go.dedis.ch/cs438/contract/parser"
)

// To record the execution state of the contract, we need to maintain
// the state tree of AST.
// It means that we are always tracking the progress of execution.
// Only focus on the types: Assumption, Condition, Ifclause, Action
type StateNode struct {
	NodeId 	  int
	Type 	  string
	Valid	  bool 	// for Assumption, Condition, Ifclause
	Executed  bool	// for Action
	Children  []*StateNode
}

func (s *StateNode) AppendChild(n *StateNode) {
	s.Children = append(s.Children, n)
}

func (s *StateNode) SetExecuted() {
	s.Executed = true
}

func (s *StateNode) GetExecuted() bool {
	return s.Executed
}

func (s *StateNode) SetValid() {
	s.Valid = true
}

func (s *StateNode) GetValid() bool {
	return s.Valid
}

func (s *StateNode) GetNodeId() int {
	return s.NodeId
}

// Construct corresponding state tree, given the code AST
// The structure of AST is rather predictable, so we don't need to recursively traverse
// We assign a id to each node, so it will be easier to retrieve & manipulate with node id
func ConstructStateTree(code_ast *parser.Code) *StateNode {
	id_counter := 0
	state_root := StateNode{id_counter, "code", false, false, []*StateNode{}}
	id_counter++
	
	// Process assumptions state
	for i := 0; i < len(code_ast.Assumptions); i++ {
		assumption_state := StateNode{id_counter, "assumption", false, false, []*StateNode{}}
		id_counter++
		condition_state := StateNode{id_counter, "condition", false, false, []*StateNode{}}
		id_counter++
		assumption_state.AppendChild(&condition_state)
		state_root.AppendChild(&assumption_state)
	}

	// Process if clauses state
	for _, ifclause := range code_ast.IfClauses {
		if_state := StateNode{id_counter, "if", false, false, []*StateNode{}}
		id_counter++
		condition_state := StateNode{id_counter, "condition", false, false, []*StateNode{}}
		id_counter++
		if_state.AppendChild(&condition_state)
		for i := 0; i < len(ifclause.Actions); i++ {
			action_state := StateNode{id_counter, "action", false, false, []*StateNode{}}
			id_counter++
			if_state.AppendChild(&action_state)
		}
		state_root.AppendChild(&if_state)
	}

	return &state_root
}

// Display the execution state AST, convenient for debug
// TODO: differientiate executed and valid in printing
func DisplayStateAST(ast parser.Code, state_ast *StateNode) string {
	root := gotree.New("State")
	output_executed := map[bool]string{true: "√", false: "x"}

	assumption_node := root.Add("Assumption")
	number_of_assumptions := len(ast.Assumptions)
	for i, assumption := range ast.Assumptions {
		assumption_node.Add(
			strconv.Itoa(state_ast.Children[i].NodeId) + ": " + 
			assumption.Condition.ToString() + 
			" [" + output_executed[state_ast.Children[i].Executed] + "]")
	}

	for i, ifclause := range ast.IfClauses {
		if_node := root.Add("If Clause")
		if_state_node := state_ast.Children[number_of_assumptions + i]

		if_node.Add(
			strconv.Itoa(if_state_node.Children[0].NodeId) + ": " + 
			ifclause.Condition.ToString() +
			" [" + output_executed[if_state_node.Children[0].Executed] + "]")
		for j, action := range ifclause.Actions {
			if_node.Add(
				strconv.Itoa(if_state_node.Children[j+1].NodeId) + ": " + 
				action.ToString() + 
				" [" + output_executed[if_state_node.Children[j+1].Executed] + "]")
		}
	}

	AST_out := root.Print()
	return AST_out
}
