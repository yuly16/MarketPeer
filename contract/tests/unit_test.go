package tests

import (
	"fmt"
	// "reflect"
	// "encoding/json"
	// "errors"
	// "time"

	"github.com/alecthomas/participle/v2"
	"testing"
	"github.com/stretchr/testify/require"
	// "go.dedis.ch/cs438/contract"
	"go.dedis.ch/cs438/contract/impl"
)

// Unit tests of the Smart contract functionalities

// Test parsing Value
func Test_Parser_Value(t *testing.T) {
	value_strings := []string{`"parser"`, `"Hello World"`}
	value_floats := []string{"0", "42", "123456", "0.125", "1.56789"}

	expected_strings := []string{`parser`, `Hello World`}
	expected_floats := []float64{0, 42, 123456, 0.125, 1.56789}

	var ValueParser = participle.MustBuild(&impl.Value{},
		participle.Lexer(impl.SMTLexer),
		participle.Unquote("String"),
	)

	var parsed_strings []string
	var parsed_floats []float64

	for _, s := range value_strings {
		val_ast := &impl.Value{}
		err := ValueParser.ParseString("", s, val_ast)
		require.NoError(t, err)
		parsed_strings = append(parsed_strings, *val_ast.String)
	}
	require.Equal(t, expected_strings, parsed_strings)

	for _, s := range value_floats {
		val_ast := &impl.Value{}
		err := ValueParser.ParseString("", s, val_ast)
		require.NoError(t, err)
		parsed_floats = append(parsed_floats, *val_ast.Number)
	}
	require.Equal(t, expected_floats, parsed_floats)

}

// Test parsing Object
func Test_Parser_Object(t *testing.T) {
	object_strings := []string{
		`buyer.balance`,
		`buyer.storage.attribute`,
		`seller.storage.product.amount`,
	}

	expected_objects := []*impl.Object{
		{
			Role: "buyer",
			Fields: []*impl.Field{
				{Name: "balance"},
			},
		},
		{
			Role: "buyer",
			Fields: []*impl.Field{
				{Name: "storage"},
				{Name: "attribute"},
			},
		},
		{
			Role: "seller",
			Fields: []*impl.Field{
				{Name: "storage"},
				{Name: "product"},
				{Name: "amount"},
			},
		},
	}

	var ObjectParser = participle.MustBuild(&impl.Object{},
		participle.Lexer(impl.SMTLexer),
		participle.Unquote("String"),
	)

	var parsed_objects []*impl.Object

	for _, s := range object_strings {
		obj_ast := &impl.Object{}
		err := ObjectParser.ParseString("", s, obj_ast)
		require.NoError(t, err)
		parsed_objects = append(parsed_objects, obj_ast)
	}
	require.Equal(t, expected_objects, parsed_objects)

}

// Test parsing Condition
func Test_Parser_Condition(t *testing.T) {
	condition_strings := []string{
		`buyer.balance > 10.5`,
		`buyer.storage.attribute == "available"`,
		`seller.storage.product.amount <= 5`,
	}
	expected_value1 := float64(10.5)
	expected_value2 := "available"
	expected_value3 := float64(5)

	expected_conditions := []*impl.Condition{
		{
			Object: impl.Object{
				Role: "buyer",
				Fields: []*impl.Field{
					{Name: "balance"},
				},
			},
			Op: ">",
			Value: impl.Value{
				String: nil,
				Number: &expected_value1,
			},
		},
		{
			Object: impl.Object{
				Role: "buyer",
				Fields: []*impl.Field{
					{Name: "storage"},
					{Name: "attribute"},
				},
			},
			Op: "==",
			Value: impl.Value{
				String: &expected_value2,
				Number: nil,
			},
		},
		{
			Object: impl.Object{
				Role: "seller",
				Fields: []*impl.Field{
					{Name: "storage"},
					{Name: "product"},
					{Name: "amount"},
				},
			},
			Op: "<=",
			Value: impl.Value{
				String: nil,
				Number: &expected_value3,
			},
		},
	}

	var ConditionParser = participle.MustBuild(&impl.Condition{},
		participle.Lexer(impl.SMTLexer),
		participle.Unquote("String"),
	)

	var parsed_conditions []*impl.Condition

	for _, s := range condition_strings {
		cond_ast := &impl.Condition{}
		err := ConditionParser.ParseString("", s, cond_ast)
		require.NoError(t, err)
		parsed_conditions = append(parsed_conditions, cond_ast)
	}
	require.Equal(t, expected_conditions, parsed_conditions)

}

// Test parsing Action
func Test_Parser_Action(t *testing.T) {
	action_strings := []string{
		`buyer.transfer("seller_id", 1.25)`,
		`seller.send("seller_id", "product_id", 50)`,
	}
	expected_value1_1 := "seller_id"
	expected_value1_2 := float64(1.25)
	expected_value2_1 := "seller_id"
	expected_value2_2 := "product_id"
	expected_value2_3 := float64(50)

	expected_actions := []*impl.Action{
		{
			Role: "buyer",
			Action: "transfer",
			Params: []*impl.Value{
				{String: &expected_value1_1, Number: nil},
				{String: nil, Number: &expected_value1_2},
			},
		},
		{
			Role: "seller",
			Action: "send",
			Params: []*impl.Value{
				{String: &expected_value2_1, Number: nil},
				{String: &expected_value2_2, Number: nil},
				{String: nil, Number: &expected_value2_3},
			},
		},
	}
	
	var ActionParser = participle.MustBuild(&impl.Action{},
		participle.Lexer(impl.SMTLexer),
		participle.Unquote("String"),
	)

	var parsed_actions []*impl.Action

	for _, s := range action_strings {
		action_ast := &impl.Action{}
		err := ActionParser.ParseString("", s, action_ast)
		require.NoError(t, err)
		parsed_actions = append(parsed_actions, action_ast)
	}
	require.Equal(t, expected_actions, parsed_actions)

}

// Parsing contract code only with one assumption
func Test_Parser_Assume(t *testing.T) {

	assume_strings := []string{
		`ASSUME buyer.balance > 10.5`,
		`ASSUME buyer.storage.attribute == "available"`,
	}
	expected_value1 := float64(10.5)
	expected_value2 := "available"

	expected_assume := []*impl.Assumption{
		{
			Condition: impl.Condition{
				Object: impl.Object{
					Role: "buyer",
					Fields: []*impl.Field{
						{Name: "balance"},
					},
				},
				Op: ">",
				Value: impl.Value{
					String: nil,
					Number: &expected_value1,
				},
			},
		},
		{
			Condition: impl.Condition{
				Object: impl.Object{
					Role: "buyer",
					Fields: []*impl.Field{
						{Name: "storage"},
						{Name: "attribute"},
					},
				},
				Op: "==",
				Value: impl.Value{
					String: &expected_value2,
					Number: nil,
				},
			},
		},
	}

	var AssumeParser = participle.MustBuild(&impl.Assumption{},
		participle.Lexer(impl.SMTLexer),
		participle.Unquote("String"),
	)

	var parsed_assume []*impl.Assumption

	for _, s := range assume_strings {
		assume_ast := &impl.Assumption{}
		err := AssumeParser.ParseString("", s, assume_ast)
		require.NoError(t, err)
		parsed_assume = append(parsed_assume, assume_ast)
	}
	require.Equal(t, expected_assume, parsed_assume)

}

// Parsing contract code with one if clause
func Test_Parser_Ifclause(t *testing.T) {

	if_strings := []string{
		`IF buyer.balance > 10.5 THEN
		 	buyer.transfer("seller_id", 1.25)
			seller.send("seller_id", "product_id", 50)
		`,
	}
	expected_value1 := float64(10.5)
	expected_value2 := "seller_id"
	expected_value3 := float64(1.25)
	expected_value4 := "product_id"
	expected_value5 := float64(50)

	expected_if := []*impl.IfClause{
		{
			Condition: impl.Condition{
				Object: impl.Object{
					Role: "buyer",
					Fields: []*impl.Field{
						{Name: "balance"},
					},
				},
				Op: ">",
				Value: impl.Value{
					String: nil,
					Number: &expected_value1,
				},
			},
			Actions: []*impl.Action{
				{
					Role: "buyer",
					Action: "transfer",
					Params: []*impl.Value{
						{String: &expected_value2, Number: nil},
						{String: nil, Number: &expected_value3},
					},
				},
				{
					Role: "seller",
					Action: "send",
					Params: []*impl.Value{
						{String: &expected_value2, Number: nil},
						{String: &expected_value4, Number: nil},
						{String: nil, Number: &expected_value5},
					},
				},
			},
		},
	}

	var IfParser = participle.MustBuild(&impl.IfClause{},
		participle.Lexer(impl.SMTLexer),
		participle.Unquote("String"),
	)

	var parsed_if []*impl.IfClause

	for _, s := range if_strings {
		if_ast := &impl.IfClause{}
		err := IfParser.ParseString("", s, if_ast)
		require.NoError(t, err)
		parsed_if = append(parsed_if, if_ast)
	}
	require.Equal(t, expected_if, parsed_if)

}

// Parsing complete contract code case 
func Test_Parser_Case(t *testing.T) {

	code_strings := []string{
		`
		ASSUME seller.balance > 100
		ASSUME seller.product.amount != 0
		IF buyer.balance > 10.5 THEN
		 	buyer.transfer("seller_id", 1.25)
			seller.send("seller_id", "product_id", 50)
		`,
	}
	expected_value1 := float64(100)
	expected_value2 := float64(0)
	expected_value3 := float64(10.5)
	expected_value4 := "seller_id"
	expected_value5 := float64(1.25)
	expected_value6 := "product_id"
	expected_value7 := float64(50)

	expected_code := []*impl.Code{
		{
			Assumptions: []*impl.Assumption{
				{
					Condition: impl.Condition{
						Object: impl.Object{
							Role: "seller",
							Fields: []*impl.Field{
								{Name: "balance"},
							},
						},
						Op: ">",
						Value: impl.Value{
							String: nil,
							Number: &expected_value1,
						},
					},
				},
				{
					Condition: impl.Condition{
						Object: impl.Object{
							Role: "seller",
							Fields: []*impl.Field{
								{Name: "product"},
								{Name: "amount"},
							},
						},
						Op: "!=",
						Value: impl.Value{
							String: nil,
							Number: &expected_value2,
						},
					},
				},
			},
			IfClauses: []*impl.IfClause{
				{
					Condition: impl.Condition{
						Object: impl.Object{
							Role: "buyer",
							Fields: []*impl.Field{
								{Name: "balance"},
							},
						},
						Op: ">",
						Value: impl.Value{
							String: nil,
							Number: &expected_value3,
						},
					},
					Actions: []*impl.Action{
						{
							Role: "buyer",
							Action: "transfer",
							Params: []*impl.Value{
								{String: &expected_value4, Number: nil},
								{String: nil, Number: &expected_value5},
							},
						},
						{
							Role: "seller",
							Action: "send",
							Params: []*impl.Value{
								{String: &expected_value4, Number: nil},
								{String: &expected_value6, Number: nil},
								{String: nil, Number: &expected_value7},
							},
						},
					},
				},
			},
		},
	}

	var parsed_code []*impl.Code
	for _, s := range code_strings {
		code_ast, err := impl.Parse(s)
		require.NoError(t, err)
		parsed_code = append(parsed_code, &code_ast)
	}
	require.Equal(t, expected_code, parsed_code)

}

// Marshal and Unmarshal the contract instance
func Test_Contract_Marshal(t *testing.T) {
	
	contract_code := 
	`
		ASSUME seller.balance > 100
		ASSUME seller.product.amount != 0
		IF buyer.balance > 10.5 THEN
		 	buyer.transfer("seller_id", 1.25)
			seller.send("seller_id", "product_id", 50)
	`
	
	// create a contract instance
	contract_inst := impl.NewContract(
		"1", // contract_id
		"Two-party commodity purchase contract", // contract_name
		contract_code, // plain_code
		"127.0.0.1", // proposer_addr
		"00000001", // proposer_account
		"127.0.0.2", // acceptor_addr
		"00000002", // acceptor_account
	)

	// marshal & unmarshal reverse check
	fmt.Println(contract_inst.String())
	buf, err := contract_inst.Marshal()
	require.NoError(t, err)
	
	var contract_unmarshal impl.Contract
	err = impl.Unmarshal(buf, &contract_unmarshal)
	require.NoError(t, err)
	fmt.Println(contract_unmarshal.String())
	
}

func Test_Debug(t *testing.T) {

	contract_code := 
	`
		ASSUME seller.balance > 100
		ASSUME seller.product.amount != 0
		IF buyer.balance > 10.5 THEN
		 	buyer.transfer("seller_id", 1.25)
			seller.send("seller_id", "product_id", 50)
	`

	// create a contract instance
	contract_inst := impl.NewContract(
		"1", // contract_id
		"Two-party commodity purchase contract", // contract_name
		contract_code, // plain_code
		"127.0.0.1", // proposer_addr
		"00000001", // proposer_account
		"127.0.0.2", // acceptor_addr
		"00000002", // acceptor_account
	)

	code_ast, err := impl.Parse(contract_code)
	require.NoError(t, err)

	fmt.Println(contract_inst.String())
	fmt.Println(impl.DisplayAST(code_ast))
}


