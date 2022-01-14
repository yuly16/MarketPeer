package impl

import (
	"github.com/alecthomas/participle/v2" // lib for building the parser
	"github.com/alecthomas/participle/v2/lexer" // lib for building the lexer
)

// This module parse the smart contract code from plain text to internal
// data structure

// Smart contract Specification: 
// 2 kind of roles: proposer (buyer), acceptor (seller) (maintained in Contract instance)
// 3 kind of primitives: Assume, If, Transaction Action (supported API of blockchain)
// Expression supported: number, role.attribute, role.productID.attribute, role.action(params) & 1 line for 1 expr
// Condition supported: role.(productID).attribute >/</=/>=/<=/!= number (先实现个强约束的)
// Revert: not explicitly specified, but canbe invertedly executed (refund就是action的逆操作)
// 交给Client端去发起revoke call，把已经进行的操作撤销
// How to execute: 

// Example:
// ASSUME seller.balance > 5
// IF seller.product0.price <= 1.5 THEN
// 		buyer.transfer("123", 3) # sellerID, money
// 		seller.send("456", "0001", 2) # sellerID, productID, amount
// (SUCCESS)

// Lexer for the contract code. Rules are specified with regexp.
// Need to tokenize to the minimum unit before parsing
// nil meaning that the lexer is simple/stateless.
var SMTLexer = lexer.MustSimple([]lexer.Rule{
	{`Keyword`, `(?i)\b(ASSUME|IF|THEN)\b`, nil}, // not case sensitive
	{`Ident`, `[a-zA-Z][a-zA-Z0-9_]*`, nil},
	{`String`, `"(.*?)"`, nil}, // quoted string tokens
	{`Float`, `\d+(?:\.\d+)?`, nil},
	{`Operator`, `==|!=|>=|<=|>|<`, nil}, // only comparison operator
	{"comment", `[#;][^\n]*`, nil},
	{"Punct", `[(),\.]`, nil},
	{"whitespace", `\s+`, nil},
})

// Specify participle grammar for contract code
type Code struct {
	Assumptions []*Assumption `@@*` // zero or more assumptions
	IfClauses	[]*IfClause   `@@*` // zero or more If clauses
}

type Assumption struct { // each assumption specifies a condition
	Condition  Condition `( "ASSUME" @@ )`
}

type Condition struct {
	Object Object  `@@`
	Op 	   string `@Operator`
	Value  Value	 `@@`
}

type IfClause struct { // one condition with one or more actions
	Condition  Condition  `"IF" @@`
	Actions    []*Action  `( "THEN" @@+ )`
}

type Object struct {
	Role 	string 		`( @"buyer" | @"seller" )`
 	Fields	[]*Field 	`@@*`
}

type Field struct {
	Name  string  `"." @Ident`
}

type Action struct {
	Role 	string   ` ( @"buyer" | @"seller" )`
	Action	string   ` ( "." (@"transfer" | @"send") )` // TODO: Action to be all blockchain primitives supported
	Params  []*Value `( "(" ( @@ ( "," @@ )* )? ")" )`
}

type Value struct {
	String *string `@String`
	Number *float64 `| @Float`
}

// parser for contract code
var SMTParser = participle.MustBuild(&Code{},
	participle.Lexer(SMTLexer),
	participle.Unquote("String"),
)

// Parsing for the contract code
func Parse(plain_code string) (Code, error){
	ast := &Code{}
	err := SMTParser.ParseString("", plain_code, ast)
	return *ast, err
}



