package types

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"go.dedis.ch/cs438/transport"
)

// -----------------------------------------------------------------------------
// ChatMessage

// NewEmpty implements types.Message.
func (c ChatMessage) NewEmpty() Message {
	return &ChatMessage{}
}

func ChatMsgCallback(msg Message, pkt transport.Packet) error {
	log.Info().Str("chat msg", msg.String())
	return nil
}

// Name implements types.Message.
// NOTE: it actually acts like a *static* method in other languages
func (ChatMessage) Name() string {
	return "chat"
}

// String implements types.Message.
func (c ChatMessage) String() string {
	return fmt.Sprintf("<%s>", c.Message)
}

// HTML implements types.Message.
func (c ChatMessage) HTML() string {
	return c.String()
}
