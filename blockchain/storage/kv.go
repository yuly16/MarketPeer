package storage

import (
	"errors"
)

var ErrKeyNotFound error = errors.New("key not found")
var ErrValueNotMatch error = errors.New("value not match")

type Copyable interface {
	Copy() Copyable
}

type KV interface {
	// methods as a basic key-value mapping
	Get(key string) (interface{}, error)
	Put(key string, value interface{}) error
	Del(key string) error
	For(func(key string, value interface{}) error) error
	Copy() KV
	Hash() string
}

type KVFactory func() KV

func CreateSimpleKV() KV {
	return NewSimpleKV()
}

func CreateMerkleKV() KV {
	panic(errors.New("not implemented"))
}
