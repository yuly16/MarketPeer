package types

import (
	"fmt"
	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/blockchain/storage"
)

// -----------------------------------------------------------------------------
// BlockMessage

// NewEmpty implements types.Message.
func (c BlockMessage) NewEmpty() Message {
	// FIXME: merkle tree will not work here
	return &BlockMessage{Block: *block.NewBlockBuilder(storage.CreateSimpleKV).Build()}
}

// Name implements types.Message.
func (c BlockMessage) Name() string {
	return "blockmessage"
}

// String implements types.Message.
func (c BlockMessage) String() string {
	return c.Block.String()
}

// HTML implements types.Message.
func (c BlockMessage) HTML() string {
	return c.String()
}

// -----------------------------------------------------------------------------
// AskForBlockMessage

// NewEmpty implements types.Message.
func (c AskForBlockMessage) NewEmpty() Message {
	// FIXME: merkle tree will not work here
	return &AskForBlockMessage{}
}

// Name implements types.Message.
func (c AskForBlockMessage) Name() string {
	return "AskForBlockMessage"
}

// String implements types.Message.
func (c AskForBlockMessage) String() string {
	return fmt.Sprintf("AskForBlockMessage: From=%s, timestamp=%d, number=%d", c.From, c.TimeStamp, c.Number)
}

// HTML implements types.Message.
func (c AskForBlockMessage) HTML() string {
	return c.String()
}

// -----------------------------------------------------------------------------
// AskForBlockMessage

// NewEmpty implements types.Message.
func (c AskForBlockReplyMessage) NewEmpty() Message {
	// FIXME: merkle tree will not work here
	return &AskForBlockReplyMessage{}
}

// Name implements types.Message.
func (c AskForBlockReplyMessage) Name() string {
	return "AskForBlockReplyMessage"
}

// String implements types.Message.
func (c AskForBlockReplyMessage) String() string {
	return fmt.Sprintf("AskForBlockReplyMessage: timestamp=%d, block=%s", c.TimeStamp, c.Block.String())
}

// HTML implements types.Message.
func (c AskForBlockReplyMessage) HTML() string {
	return c.String()
}
