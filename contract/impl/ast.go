package impl

import (
	"github.com/disiqueira/gotree" // lib for print tree structure in terminal
)

// Combination of code AST and state AST, provides API for retrieval and validfy
type AST struct {
	Code_tree	Code
	State_tree	*StateNode
}

// // 
// func (ast *AST) () {

// }

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
