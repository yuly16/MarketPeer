package impl

import (
	// "fmt"
	"strings"
    "encoding/json"

	"go.dedis.ch/cs438/contract"
)

// implements contract.ContractCode, maintained in contract account
type Contract struct {
	contract.SmartContract
	Contract_id 	 string
	Contract_name	 string
	Code_ast 	 	 Code
	Code_plain		 string
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
	
	code, err := Parse(plain_code)
	if err != nil {
		panic(err)
	}

	return &Contract{
		Contract_id: contract_id,
		Contract_name: contract_name,
		Code_ast: code,
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
func (c *Contract) Unmarshal(data []byte) error {
	return json.Unmarshal(data, c)
}

// Contract.String() outputs the contract in readable format
func (c Contract) String() string {
	out := new(strings.Builder)

	out.WriteString("=================================\n")
	out.WriteString("| Contract: " + c.Contract_name + "\n")
	out.WriteString("| ID: " + c.Contract_id + "\n")
	out.WriteString("=================================\n")
	out.WriteString("| Proposer: [" + c.Proposer_addr + "] (" + c.Proposer_account + ")\n")
	out.WriteString("| Acceptor: [" + c.Acceptor_addr + "] (" + c.Acceptor_account + ")\n")
	out.WriteString("=================================\n")
	out.WriteString("| Contract code: " + "\n")
	out.WriteString(c.Code_plain)
	out.WriteString("=================================\n")

	return out.String()
}

