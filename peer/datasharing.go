package peer

import (
	"fmt"
	"io"
	"regexp"
	"time"
)

// MetafileSep defines the separation between chunk hashes in the metafile.
const MetafileSep = "\n"

// DataSharing describes functions to share data in a bittorrent-like system.
type DataSharing interface {
	// Upload stores a new data blob on the peer and will make it available to
	// other peers. The blob will be split into chunks.
	//
	// - Implemented in HW2
	Upload(data io.Reader) (metahash string, err error)

	// Download will get all the necessary chunks corresponding to the given
	// metahash that references a blob, and return a reconstructed blob. The
	// peer will save locally the chunks that it doesn't have for further
	// sharing. Returns an error if it can't get the necessary chunks.
	//
	// - Implemented in HW2
	Download(metahash string) ([]byte, error)

	// Tag creates a mapping between a (file)name and a metahash.
	//
	// - Implemented in HW2
	// - Improved in HW3: ensure uniqueness with blockchain/TLC/Paxos
	Tag(name string, mh string) error

	// Resolve returns the corresponding metahash of a given (file)name. Returns
	// an empty string if not found.
	//
	// - Implemented in HW2
	Resolve(name string) (metahash string)

	// GetCatalog returns the peer's catalog. See below for the definition of a
	// catalog.
	//
	// - Implemented in HW2
	GetCatalog() Catalog

	// UpdateCatalog tells the peer about a piece of data referenced by 'key'
	// being available on other peers. It should update the peer's catalog. See
	// below for the definition of a catalog.
	//
	// - Implemented in HW2
	UpdateCatalog(key string, peer string)

	// SearchAll returns all the names that exist matching the given regex. It
	// merges results from the local storage and from the search request reply
	// sent to a random neighbor using the provided budget. It makes the peer
	// update its catalog and name storage according to the SearchReplyMessages
	// received. Returns an empty result if nothing found. An error is returned
	// in case of an exceptional event.
	//
	// - Implemented in HW2
	SearchAll(reg regexp.Regexp, budget uint, timeout time.Duration) (names []string, err error)

	// SearchFirst uses an expanding ring configuration and returns a name as
	// soon as it finds a peer that "fully matches" a data blob. It makes the
	// peer update its catalog and name storage according to the
	// SearchReplyMessages received. Returns an empty string if nothing was
	// found.
	SearchFirst(pattern regexp.Regexp, conf ExpandingRing) (name string, err error)

	// Store stores a key value into chord system.
	// - Implemented in final project
	Store(key string) (err error)

	// Join adds current node into chord system containing a node name
	Join(member string) (err error)

	// Init inits chord network
	Init(member string)

	// PrintInfo is used for debug
	PrintInfo() uint

	// Lookup returns the ip address that is responsible for this key
	Lookup(key string) (string, error)

	// Lookup returns the ip address that is responsible for this key
	LookupHashId(key uint) (uint, error)

	// GetId acquire the value of key
	GetId(key uint) (interface{}, bool)

	// PutId puts key into current node
	PutId(key uint, data interface{})

	// Get acquire the value of key
	Get(key string) (interface{}, bool)

	// Put puts key into current node
	Put(key string, data interface{})

	// GetChordStorage outputs chordStorage
	GetChordStorage() map[uint]interface{}

	GetFingerTable() []uint

	GetPredecessor() uint

	GetSuccessor() uint

	GetChordId() uint
}

// Catalog tells, for a given piece of data referenced by a key, a bag of peers
// that can provide this piece of data. For example:
//   {
//     "aef123": {
//       "127.0.0.1:3": {}, "127.0.0.1:2": {}
//     },
//     ...
//   }
// tells that the piece of data with key "aef123" is available at peers whose
// addresses are "127.0.0.1:3" and "127.0.0.1:2".
//
// Elements stored by a peer must not appear in its own catalog. The peer uses
// the blob storage for that.
type Catalog map[string]map[string]struct{}

// ExpandingRing defines an expanding ring configuration.
type ExpandingRing struct {
	// Initial budget. Should be at least 1.
	Initial uint

	// Budget is multiplied by factor after each try
	Factor uint

	// Number of times to try. A value of 1 means there will be only 1 attempt.
	Retry uint

	// Timeout before retrying when no response received.
	Timeout time.Duration
}

func (ring ExpandingRing) String() string {
	return fmt.Sprintf("{Initial=%d, factor=%d, retry=%d, timeout=%v}", ring.Initial, ring.Factor, ring.Retry, ring.Timeout)
}
