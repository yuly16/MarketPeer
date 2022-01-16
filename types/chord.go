package types

import "fmt"

// -----------------------------------------------------------------------------
// ChordFindSuccessorMessage

// NewEmpty implements types.Message.
func (c ChordFindSuccessorMessage) NewEmpty() Message {
	return &ChordFindSuccessorMessage{}
}

// Name implements types.Message.
func (c ChordFindSuccessorMessage) Name() string {
	return "chordfindsuccessor"
}

// String implements types.Message.
func (c ChordFindSuccessorMessage) String() string {
	return fmt.Sprintf("{ChordFindSuccessorMessage %s - %d}", c.Source, c.ID)
}

// HTML implements types.Message.
func (c ChordFindSuccessorMessage) HTML() string {
	return c.String()
}

// -----------------------------------------------------------------------------
// ChordFindSuccessorReplyMessage

// NewEmpty implements types.Message.
func (c ChordFindSuccessorReplyMessage) NewEmpty() Message {
	return &ChordFindSuccessorReplyMessage{}
}

// Name implements types.Message.
func (c ChordFindSuccessorReplyMessage) Name() string {
	return "chordfindsuccessorreply"
}

// String implements types.Message.
func (c ChordFindSuccessorReplyMessage) String() string {
	return fmt.Sprintf("{ChordFindSuccessorReplyMessage %s - %d}", c.Dest, c.ID)
}

// HTML implements types.Message.
func (c ChordFindSuccessorReplyMessage) HTML() string {
	return c.String()
}

// -----------------------------------------------------------------------------
// ChordAskPredecessorMessage

// NewEmpty implements types.Message.
func (c ChordAskPredecessorMessage) NewEmpty() Message {
	return &ChordAskPredecessorMessage{}
}

// Name implements types.Message.
func (c ChordAskPredecessorMessage) Name() string {
	return "chordaskpredecessormessage"
}

// String implements types.Message.
func (c ChordAskPredecessorMessage) String() string {
	return fmt.Sprintf("{ChordAskPredecessorMessage}")
}

// HTML implements types.Message.
func (c ChordAskPredecessorMessage) HTML() string {
	return c.String()
}

// -----------------------------------------------------------------------------
// ChordReplyPredecessorMessage

// NewEmpty implements types.Message.
func (c ChordReplyPredecessorMessage) NewEmpty() Message {
	return &ChordReplyPredecessorMessage{}
}

// Name implements types.Message.
func (c ChordReplyPredecessorMessage) Name() string {
	return "chordreplypredecessormessage"
}

// String implements types.Message.
func (c ChordReplyPredecessorMessage) String() string {
	return fmt.Sprintf("{ChordReplyPredecessorMessage %s}", c.Predecessor)
}

// HTML implements types.Message.
func (c ChordReplyPredecessorMessage) HTML() string {
	return c.String()
}

// -----------------------------------------------------------------------------
// ChordNotifyMessage

// NewEmpty implements types.Message.
func (c ChordNotifyMessage) NewEmpty() Message {
	return &ChordNotifyMessage{}
}

// Name implements types.Message.
func (c ChordNotifyMessage) Name() string {
	return "chordnotifymessage"
}

// String implements types.Message.
func (c ChordNotifyMessage) String() string {
	return fmt.Sprintf("{ChordNotifyMessage }")
}

// HTML implements types.Message.
func (c ChordNotifyMessage) HTML() string {
	return c.String()
}


// -----------------------------------------------------------------------------
// ChordTransferKeyMessage

// NewEmpty implements types.Message.
func (c ChordTransferKeyMessage) NewEmpty() Message {
	return &ChordTransferKeyMessage{}
}

// Name implements types.Message.
func (c ChordTransferKeyMessage) Name() string {
	return "chordtransferkeymessage"
}

// String implements types.Message.
func (c ChordTransferKeyMessage) String() string {
	return fmt.Sprintf("{ChordTransferKeyMessage}")
}

// HTML implements types.Message.
func (c ChordTransferKeyMessage) HTML() string {
	return c.String()
}

// -----------------------------------------------------------------------------
// ChordTransferKeyMessage

// NewEmpty implements types.Message.
func (c ChordInsertKVMessage) NewEmpty() Message {
	return &ChordInsertKVMessage{}
}

// Name implements types.Message.
func (c ChordInsertKVMessage) Name() string {
	return "chordinsertkvmessage"
}

// String implements types.Message.
func (c ChordInsertKVMessage) String() string {
	return fmt.Sprintf("{ChordInsertKVMessage}")
}

// HTML implements types.Message.
func (c ChordInsertKVMessage) HTML() string {
	return c.String()
}