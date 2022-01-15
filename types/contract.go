package types

// ProposeContractMessage sent by proposer to contract account,
// to initiate the corresponding smart contract instance.
// Include the required params to initiate a contract instance,
// except for contract id.
//
type ProposeContractMessage struct {
	Code 				string
	Contract_name 		string
	Proposer_addr		string
	Proposer_account	string
	Acceptor_addr		string
	Acceptor_account 	string
}

// RequestSignContractMessage sent by contract account to both
// parties of the contract, asking for their acknowledgement.
//
type RequestSignContractMessage struct {
	Contract_id string
}

// AcceptContractMessage sent by parties to respond to request sign
// contract message. Digital signatures need verified.
//
type AcceptContractMessage struct {
	Acked_Contract_id string
}

// ActionContractMessage sent by smart contract to activate the
// transaction action of buyer/seller.
// 
type ActionContractMessage struct {
	Contract_id string
	Action string // supported actions in blockchain
	Params  // two parameter types: string & float
}

// ActionAckContractMessage sent by buyer/seller to verify their
// execution of specified actions. (Ack for ActionContractMessage)
//
type ActionAckContractMessage struct {
	Contract_id string
	AckedPacketID string
}

// RevokeContractMessage proposed by some party in the contract and 
// revokes the execution.
// 
type RevokeContractMessage struct {
	Contract_id string
}

// ResultContractMessage sent by contract account to inform the 
// success/failure of the contract. Reason for failure will be 
// included: revoke request, some condition not satisfied.
// 
type ResultContractMessage struct {
	Contract_id string
	Status		bool
	Feedback	string
}