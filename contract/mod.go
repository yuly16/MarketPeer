package contract

import (
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/contract/parser"
)

// SmartContract describes functions to manipulate the code section in smart contract
type SmartContract interface {
	// Execute directly runs the contract code segment, 
	// and ensures the desired property.
	Execute() (error)

	String() (string)

	Marshal() ([]byte, error)

	ValidateAssumptions(storage.KV) (bool, error)

	CollectActions(storage.KV) ([]parser.Action, error)
}

