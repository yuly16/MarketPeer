package contract

import (

)

// SmartContract describes functions to manipulate the code section in smart contract
type SmartContract interface {
	// Validate ensures the validity of the contract code, 
	// some bad cases (e.g., grammartical errors) will be detected.
	Validate() (bool, error)

	// Execute directly runs the contract code segment, 
	// and ensures the desired property.
	Execute() (error)

	// // ToByteCode converts the code segment to byte code, which 
	// // stores in the smart contract account. To simplify, Bytecode
	// // will not be executed in the virtual machine, as performed 
	// // in EVM, but to be decoded to another ContractCode instance.
	// deprecated - implemented with marshal and unmarshal
	// ToByteCode() (code_byte []byte, error)

	String() (string)

	Marshal() ([]byte, error)
}
