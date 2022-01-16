package types

import (
	"fmt"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/storage"
)

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

// -----------------------------------------------------------------------------
// SyncAccountMessage

// NewEmpty implements types.Message.
func (c SyncAccountMessage) NewEmpty() Message {
	return &SyncAccountMessage{}
}

// Name implements types.Message.
func (c SyncAccountMessage) Name() string {
	return "sync-account-message"
}

// String implements types.Message.
func (c SyncAccountMessage) String() string {
	return fmt.Sprintf("{SyncAccountMessage networkAddr=%s, addr=%s, stamp=%d}",
		c.NetworkAddr, c.Addr.String()[:6]+"...", c.Timestamp)
}

// HTML implements types.Message.
func (c SyncAccountMessage) HTML() string {
	return c.String()
}

// -----------------------------------------------------------------------------
// SyncAccountReplyMessage

// NewEmpty implements types.Message.
func (c SyncAccountReplyMessage) NewEmpty() Message {
	return &SyncAccountReplyMessage{State: *account.NewStateBuilder(storage.CreateSimpleKV).Build()}
}

// Name implements types.Message.
func (c SyncAccountReplyMessage) Name() string {
	return "sync-account-reply-message"
}

// String implements types.Message.
func (c SyncAccountReplyMessage) String() string {
	return fmt.Sprintf("{SyncAccountReplyMessage state=%s}", c.State.String())
}

// HTML implements types.Message.
func (c SyncAccountReplyMessage) HTML() string {
	return c.String()
}
