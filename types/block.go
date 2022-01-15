package types

import (
	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/blockchain/storage"
)

// -----------------------------------------------------------------------------
// BlockMessage

// NewEmpty implements types.Message.
func (c BlockMessage) NewEmpty() Message {
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