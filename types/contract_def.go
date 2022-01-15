package types

import fmt

// -----------------------------------------------------------------------------
// ProposeContractMessage

// NewEmpty implements types.Message.
func (c ProposeContractMessage) NewEmpty() Message {
	return &ProposeContractMessage{}
}

// Name implements types.Message.
func (ProposeContractMessage) Name() string {
	return "propose_contract"
}

// String implements types.Message.
func (c ProposeContractMessage) String() string {
	return fmt.Sprintf("{ProposeContractMessage: %s}", c.Contract_name)
}

// HTML implements types.Message.
func (c ProposeContractMessage) HTML() string {
	return c.String()
}

// -----------------------------------------------------------------------------
// RequestSignContractMessage

// NewEmpty implements types.Message.
func (c RequestSignContractMessage) NewEmpty() Message {
	return &RequestSignContractMessage{}
}

// Name implements types.Message.
func (RequestSignContractMessage) Name() string {
	return "requestsign_contract"
}

// String implements types.Message.
func (c RequestSignContractMessage) String() string {
	return fmt.Sprintf("{RequestSignContractMessage: %s}", c.Contract_id)
}

// HTML implements types.Message.
func (c RequestSignContractMessage) HTML() string {
	return c.String()
}

// -----------------------------------------------------------------------------
// AcceptContractMessage

// NewEmpty implements types.Message.
func (c AcceptContractMessage) NewEmpty() Message {
	return &AcceptContractMessage{}
}

// Name implements types.Message.
func (AcceptContractMessage) Name() string {
	return "accept_contract"
}

// String implements types.Message.
func (c AcceptContractMessage) String() string {
	return fmt.Sprintf("{AcceptContractMessage: %s}", c.Contract_id)
}

// HTML implements types.Message.
func (c AcceptContractMessage) HTML() string {
	return c.String()
}

// -----------------------------------------------------------------------------
// ActionContractMessage

// NewEmpty implements types.Message.
func (c ActionContractMessage) NewEmpty() Message {
	return &ActionContractMessage{}
}

// Name implements types.Message.
func (ActionContractMessage) Name() string {
	return "action_contract"
}

// String implements types.Message.
func (c ActionContractMessage) String() string {
	return fmt.Sprintf("{ActionContractMessage: %s - %s}", c.Contract_id, c.Action)
}

// HTML implements types.Message.
func (c ActionContractMessage) HTML() string {
	return c.String()
}

// -----------------------------------------------------------------------------
// ActionAckContractMessage

// NewEmpty implements types.Message.
func (c ActionAckContractMessage) NewEmpty() Message {
	return &ActionAckContractMessage{}
}

// Name implements types.Message.
func (ActionAckContractMessage) Name() string {
	return "actionack_contract"
}

// String implements types.Message.
func (c ActionAckContractMessage) String() string {
	return fmt.Sprintf("{ActionAckContractMessage: %s}", c.Contract_id)
}

// HTML implements types.Message.
func (c ActionAckContractMessage) HTML() string {
	return c.String()
}

// -----------------------------------------------------------------------------
// RevokeContractMessage

// NewEmpty implements types.Message.
func (c RevokeContractMessage) NewEmpty() Message {
	return &RevokeContractMessage{}
}

// Name implements types.Message.
func (RevokeContractMessage) Name() string {
	return "revoke_contract"
}

// String implements types.Message.
func (c RevokeContractMessage) String() string {
	return fmt.Sprintf("{RevokeContractMessage: %s}", c.Contract_id)
}

// HTML implements types.Message.
func (c RevokeContractMessage) HTML() string {
	return c.String()
}

// -----------------------------------------------------------------------------
// ResultContractMessage

// NewEmpty implements types.Message.
func (c ResultContractMessage) NewEmpty() Message {
	return &ResultContractMessage{}
}

// Name implements types.Message.
func (ResultContractMessage) Name() string {
	return "result_contract"
}

// String implements types.Message.
func (c ResultContractMessage) String() string {
	return fmt.Sprintf("{ResultContractMessage: %s}", c.Contract_id)
}

// HTML implements types.Message.
func (c ResultContractMessage) HTML() string {
	return c.String()
}
