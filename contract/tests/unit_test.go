package tests

import (
	"fmt"
	"time"
	"crypto/ecdsa"
	"testing"

	"go.dedis.ch/cs438/transport/channel"
	"go.dedis.ch/cs438/registry/standard"
	"github.com/alecthomas/participle/v2"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/contract/parser"
	"go.dedis.ch/cs438/contract/impl"
	"go.dedis.ch/cs438/blockchain"
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/transaction"
	z "go.dedis.ch/cs438/internal/testing"
)

// Unit tests of the Smart contract functionalities

// Test parsing Value
func Test_Parser_Value(t *testing.T) {
	value_strings := []string{`"parser"`, `"Hello World"`}
	value_floats := []string{"0", "42", "123456", "0.125", "1.56789"}

	expected_strings := []string{`parser`, `Hello World`}
	expected_floats := []float64{0, 42, 123456, 0.125, 1.56789}

	var ValueParser = participle.MustBuild(&parser.Value{},
		participle.Lexer(parser.SMTLexer),
		participle.Unquote("String"),
	)

	var parsed_strings []string
	var parsed_floats []float64

	for _, s := range value_strings {
		val_ast := &parser.Value{}
		err := ValueParser.ParseString("", s, val_ast)
		require.NoError(t, err)
		parsed_strings = append(parsed_strings, *val_ast.String)
	}
	require.Equal(t, expected_strings, parsed_strings)

	for _, s := range value_floats {
		val_ast := &parser.Value{}
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

	expected_objects := []*parser.Object{
		{
			Role: "buyer",
			Fields: []*parser.Field{
				{Name: "balance"},
			},
		},
		{
			Role: "buyer",
			Fields: []*parser.Field{
				{Name: "storage"},
				{Name: "attribute"},
			},
		},
		{
			Role: "seller",
			Fields: []*parser.Field{
				{Name: "storage"},
				{Name: "product"},
				{Name: "amount"},
			},
		},
	}

	var ObjectParser = participle.MustBuild(&parser.Object{},
		participle.Lexer(parser.SMTLexer),
		participle.Unquote("String"),
	)

	var parsed_objects []*parser.Object

	for _, s := range object_strings {
		obj_ast := &parser.Object{}
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

	expected_conditions := []*parser.Condition{
		{
			Object: parser.Object{
				Role: "buyer",
				Fields: []*parser.Field{
					{Name: "balance"},
				},
			},
			Op: ">",
			Value: parser.Value{
				String: nil,
				Number: &expected_value1,
			},
		},
		{
			Object: parser.Object{
				Role: "buyer",
				Fields: []*parser.Field{
					{Name: "storage"},
					{Name: "attribute"},
				},
			},
			Op: "==",
			Value: parser.Value{
				String: &expected_value2,
				Number: nil,
			},
		},
		{
			Object: parser.Object{
				Role: "seller",
				Fields: []*parser.Field{
					{Name: "storage"},
					{Name: "product"},
					{Name: "amount"},
				},
			},
			Op: "<=",
			Value: parser.Value{
				String: nil,
				Number: &expected_value3,
			},
		},
	}

	var ConditionParser = participle.MustBuild(&parser.Condition{},
		participle.Lexer(parser.SMTLexer),
		participle.Unquote("String"),
	)

	var parsed_conditions []*parser.Condition

	for _, s := range condition_strings {
		cond_ast := &parser.Condition{}
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

	expected_actions := []*parser.Action{
		{
			Role: "buyer",
			Action: "transfer",
			Params: []*parser.Value{
				{String: &expected_value1_1, Number: nil},
				{String: nil, Number: &expected_value1_2},
			},
		},
		{
			Role: "seller",
			Action: "send",
			Params: []*parser.Value{
				{String: &expected_value2_1, Number: nil},
				{String: &expected_value2_2, Number: nil},
				{String: nil, Number: &expected_value2_3},
			},
		},
	}
	
	var ActionParser = participle.MustBuild(&parser.Action{},
		participle.Lexer(parser.SMTLexer),
		participle.Unquote("String"),
	)

	var parsed_actions []*parser.Action

	for _, s := range action_strings {
		action_ast := &parser.Action{}
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

	expected_assume := []*parser.Assumption{
		{
			Condition: parser.Condition{
				Object: parser.Object{
					Role: "buyer",
					Fields: []*parser.Field{
						{Name: "balance"},
					},
				},
				Op: ">",
				Value: parser.Value{
					String: nil,
					Number: &expected_value1,
				},
			},
		},
		{
			Condition: parser.Condition{
				Object: parser.Object{
					Role: "buyer",
					Fields: []*parser.Field{
						{Name: "storage"},
						{Name: "attribute"},
					},
				},
				Op: "==",
				Value: parser.Value{
					String: &expected_value2,
					Number: nil,
				},
			},
		},
	}

	var AssumeParser = participle.MustBuild(&parser.Assumption{},
		participle.Lexer(parser.SMTLexer),
		participle.Unquote("String"),
	)

	var parsed_assume []*parser.Assumption

	for _, s := range assume_strings {
		assume_ast := &parser.Assumption{}
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

	expected_if := []*parser.IfClause{
		{
			Condition: parser.Condition{
				Object: parser.Object{
					Role: "buyer",
					Fields: []*parser.Field{
						{Name: "balance"},
					},
				},
				Op: ">",
				Value: parser.Value{
					String: nil,
					Number: &expected_value1,
				},
			},
			Actions: []*parser.Action{
				{
					Role: "buyer",
					Action: "transfer",
					Params: []*parser.Value{
						{String: &expected_value2, Number: nil},
						{String: nil, Number: &expected_value3},
					},
				},
				{
					Role: "seller",
					Action: "send",
					Params: []*parser.Value{
						{String: &expected_value2, Number: nil},
						{String: &expected_value4, Number: nil},
						{String: nil, Number: &expected_value5},
					},
				},
			},
		},
	}

	var IfParser = participle.MustBuild(&parser.IfClause{},
		participle.Lexer(parser.SMTLexer),
		participle.Unquote("String"),
	)

	var parsed_if []*parser.IfClause

	for _, s := range if_strings {
		if_ast := &parser.IfClause{}
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

	expected_code := []*parser.Code{
		{
			Assumptions: []*parser.Assumption{
				{
					Condition: parser.Condition{
						Object: parser.Object{
							Role: "seller",
							Fields: []*parser.Field{
								{Name: "balance"},
							},
						},
						Op: ">",
						Value: parser.Value{
							String: nil,
							Number: &expected_value1,
						},
					},
				},
				{
					Condition: parser.Condition{
						Object: parser.Object{
							Role: "seller",
							Fields: []*parser.Field{
								{Name: "product"},
								{Name: "amount"},
							},
						},
						Op: "!=",
						Value: parser.Value{
							String: nil,
							Number: &expected_value2,
						},
					},
				},
			},
			IfClauses: []*parser.IfClause{
				{
					Condition: parser.Condition{
						Object: parser.Object{
							Role: "buyer",
							Fields: []*parser.Field{
								{Name: "balance"},
							},
						},
						Op: ">",
						Value: parser.Value{
							String: nil,
							Number: &expected_value3,
						},
					},
					Actions: []*parser.Action{
						{
							Role: "buyer",
							Action: "transfer",
							Params: []*parser.Value{
								{String: &expected_value4, Number: nil},
								{String: nil, Number: &expected_value5},
							},
						},
						{
							Role: "seller",
							Action: "send",
							Params: []*parser.Value{
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

	var parsed_code []*parser.Code
	for _, s := range code_strings {
		code_ast, err := parser.Parse(s)
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

// Test state tree by printing out AST and State AST
func Test_Contract_State_Tree(t *testing.T) {

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

	code_ast, err := parser.Parse(contract_code)
	state_ast := impl.ConstructStateTree(&code_ast)
	require.NoError(t, err)

	fmt.Println(contract_inst.String())
	fmt.Println(impl.DisplayAST(code_ast))
	fmt.Println(impl.DisplayStateAST(code_ast, state_ast))
}

// Test ValidateAssumptions functionality
func Test_Contract_Execution_Condition(t *testing.T) {

	acc1, _ := accountFactory(105, "orange", 10) // buyer
	acc2, _ := accountFactory(200, "location", "Lausanne") // seller
	acc3, _ := accountFactory(200, "orange", 10) // buyer
	acc4, _ := accountFactory(100, "apple", 0) // buyer

	acc1_addr, acc2_addr := acc1.GetAddr().String(), acc2.GetAddr().String()
	acc3_addr, acc4_addr := acc3.GetAddr().String(), acc4.GetAddr().String()

	// construct world state storage
	worldState_1 := storage.CreateSimpleKV()
	worldState_1.Put(acc1_addr, acc1.GetState())
	worldState_1.Put(acc2_addr, acc2.GetState())

	worldState_2 := storage.CreateSimpleKV()
	worldState_2.Put(acc3_addr, acc3.GetState())
	worldState_2.Put(acc4_addr, acc4.GetState())
	
	// case 1: condition met
	contract_code_1 := 
	`
		ASSUME buyer.balance > 100
		ASSUME seller.location == "Lausanne"
	`
	contract_inst_1 := impl.NewContract(
		"1", // contract_id
		"Test assumption contract only", // contract_name
		contract_code_1, // plain_code
		"127.0.0.1", // proposer_addr
		acc1_addr, // proposer_account
		"127.0.0.2", // acceptor_addr
		acc2_addr, // acceptor_account
	)

	// case 2: condition not met
	contract_code_2 := 
	`
		ASSUME buyer.balance > 100
		ASSUME seller.apple > 0
	`
	contract_inst_2 := impl.NewContract(
		"1", // contract_id
		"Test assumption contract only", // contract_name
		contract_code_2, // plain_code
		"127.0.0.1", // proposer_addr
		acc3_addr, // proposer_account
		"127.0.0.2", // acceptor_addr
		acc4_addr, // acceptor_account
	)

	// validate the assumptions
	valid_1, err_1 := contract_inst_1.ValidateAssumptions(worldState_1)
	require.NoError(t, err_1)
	require.Equal(t, true, valid_1)

	valid_2, err_2 := contract_inst_2.ValidateAssumptions(worldState_2)
	require.NoError(t, err_2)
	require.Equal(t, false, valid_2)

}

// Test CollectActions functionality
func Test_Contract_Execution_IfClause(t *testing.T) {
	
	acc1, _ := accountFactory(50, "orange", 10) // buyer
	acc2, _ := accountFactory(200, "apple", 50) // seller
	acc1_addr, acc2_addr := acc1.GetAddr().String(), acc2.GetAddr().String()

	// construct world state storage
	worldState := storage.CreateSimpleKV()
	worldState.Put(acc1_addr, acc1.GetState())
	worldState.Put(acc2_addr, acc2.GetState())

	contract_code := 
	`
		IF buyer.balance > 10.5 THEN
		 	buyer.transfer("seller_account", 1.25)
			seller.send("buyer_account", "apple", 10)
		IF buyer.balance > 100 THEN
		 	buyer.transfer("seller_account", 100)
			seller.send("seller_account", "product_id", 999)
	`
	contract_inst := impl.NewContract(
		"1", // contract_id
		"Test collect actions from if clauses", // contract_name
		contract_code, // plain_code
		"127.0.0.1", // proposer_addr
		acc1_addr, // proposer_account
		"127.0.0.2", // acceptor_addr
		acc2_addr, // acceptor_account
	)

	expected_value1 := "seller_account"
	expected_value2 := float64(1.25)
	expected_value3 := "buyer_account"
	expected_value4 := "apple"
	expected_value5 := float64(10)

	expected_actions := []parser.Action{
		{
			Role: "buyer",
			Action: "transfer",
			Params: []*parser.Value{
				{String: &expected_value1, Number: nil},
				{String: nil, Number: &expected_value2},
			},
		},
		{
			Role: "seller",
			Action: "send",
			Params: []*parser.Value{
				{String: &expected_value3, Number: nil},
				{String: &expected_value4, Number: nil},
				{String: nil, Number: &expected_value5},
			},
		},
	}

	actions, err := contract_inst.CollectActions(worldState)
	require.NoError(t, err)
	require.Equal(t, expected_actions, actions)
}

func Test_Contract_Execution_Network(t *testing.T) {
	transp := channel.NewTransport()
	
	acc_buyer, pri_buyer := accountFactory(105, "apple", 5)
	acc_seller, pri_seller := accountFactory(95, "apple", 10)

	// simulate the process to create contract account (from buyer's side)
	contract_code := `
		ASSUME seller.balance > 50
		IF buyer.balance > 10 THEN
		 	buyer.transfer(5)
			seller.send("apple", 5)
	`
	contract_inst := impl.NewContract(
		"1", // contract_id
		"Test contract execution", // contract_name
		contract_code, // plain_code
		"127.0.0.1", // proposer_addr
		acc_buyer.GetAddr().String(), // proposer_account
		"127.0.0.2", // acceptor_addr
		acc_seller.GetAddr().String(), // acceptor_account
	)
	contract_bytecode, err := contract_inst.Marshal()
	require.NoError(t, err)
	acc_contract, _ := contractFactory(0, contract_bytecode)

	genesisFactory := func() *block.Block {
		return generateGenesisBlock(storage.CreateSimpleKV, acc_buyer, acc_seller, acc_contract)
	}

	nodeFactory := func(acc *account.Account, pri *ecdsa.PrivateKey) *blockchain.FullNode {
		sock, err := transp.CreateSocket("127.0.0.1:0")
		require.NoError(t, err)
		fullNode, _ := z.NewTestFullNode(t,
			z.WithSocket(sock),
			z.WithMessageRegistry(standard.NewRegistry()),
			z.WithPrivateKey(pri),
			z.WithAccount(acc),
			z.WithGenesisBlock(genesisFactory()),
		)
		return fullNode
	}

	node1 := nodeFactory(acc_buyer, pri_buyer)
	node2 := nodeFactory(acc_seller, pri_seller)
	node1.AddPeer(node2)
	node2.AddPeer(node1)

	node1.Start()
	node2.Start()
	time.Sleep(200 * time.Millisecond)

	// trigger the contract from the seller side to contract account
	txn := transaction.NewTransaction(0, 0, *node2.GetAccountAddr(), *acc_contract.GetAddr())
	node2.SubmitTxn(txn)

	time.Sleep(10 * time.Second)
	fmt.Printf("node1 chain: \n%s", node1.GetChain())
	fmt.Printf("node2 chain: \n%s", node2.GetChain())

	node1.Stop()
	node2.Stop()
}