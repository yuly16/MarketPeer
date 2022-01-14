package types

import "fmt"

// -----------------------------------------------------------------------------
// WalletTransactionMessage

// NewEmpty implements types.Message.
func (c WalletTransactionMessage) NewEmpty() Message {
	return &WalletTransactionMessage{}
}

// Name implements types.Message.
func (c WalletTransactionMessage) Name() string {
	return "wallettransactionmessage"
}

// String implements types.Message.
func (c WalletTransactionMessage) String() string {
	return fmt.Sprintf("{WalletTransactionMessage}")
}

// HTML implements types.Message.
func (c WalletTransactionMessage) HTML() string {
	return c.String()
}
