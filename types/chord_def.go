package types

// ChordFindSuccessorMessage n sends this message to n',
// and n' should find the successor of ID, and tells Source
type ChordFindSuccessorMessage struct {
	// Source is the address of the peer that sends the findSuccessor
	Source string
	ID uint
}

// ChordFindSuccessorReplyMessage n' sends this message to Source,
// tells Source that Dest is the successor of ID
type ChordFindSuccessorReplyMessage struct {
	// Dest is the address of the peer that is the successor of ID
	Dest string
	ID uint
}

// ChordAskPredecessorMessage n sends this message to n',
// means n wants to know Predecessor of n'
type ChordAskPredecessorMessage struct {
}

// ChordReplyPredecessorMessage n' sends this message to n,
// means n' tells n its predecessor
type ChordReplyPredecessorMessage struct {
	Predecessor string
}

// ChordNotifyMessage n sends this message to n',
// means n tells n' n is the possible predecessor of n'
type ChordNotifyMessage struct {
}


// ChordTransferKeyMessage n sends this message to n',
// and n' should transfer the key belonging to [n, n')
type ChordTransferKeyMessage struct {
	Data map[uint]interface{}
}

// ChordInsertKVMessage n sends this message to n',
// n' should save this kv pair into its own storage.
type ChordInsertKVMessage struct {
	Key uint
	Value interface{}
}
