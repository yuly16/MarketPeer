package storage

import (
	"errors"
)

var ErrKeyNotFound error = errors.New("key not found")
var ErrValueNotMatch error = errors.New("value not match")

type KV interface {
	// methods as a basic key-value mapping
	Get(key string) (interface{}, error)
	Put(key string, value interface{}) error
	Del(key string) error
	Hash() string
}

type KVFactory func() KV

func CreateSimpleKV() KV {
	return NewSimpleKV()
}

func CreateMerkleKV() KV {
	panic(errors.New("not implemented"))
}
