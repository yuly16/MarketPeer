package storage

import (
	"crypto"
	"encoding/hex"
	"errors"
	"fmt"
)

var ErrKeyNotFound error = errors.New("key not found")
var ErrValueNotMatch error = errors.New("value not match")

type KVFactory func() KV

type KV interface {
	// methods as a basic key-value mapping
	Get(key []byte) ([]byte, error)
	Put(key []byte, value []byte) error
	Del(key []byte, value []byte) error
	Hash() string
}

type SimpleKV struct {
	kv map[string][]byte
}

func (skv *SimpleKV) Get(key []byte) ([]byte, error) {
	value, ok := skv.kv[string(key)]
	if !ok {
		return nil, ErrKeyNotFound
	}
	return value, nil
}

func (skv *SimpleKV) Put(key []byte, value []byte) error {
	skv.kv[string(key)] = value
	return nil
}

func (skv *SimpleKV) Del(key []byte, value []byte) error {
	actual, ok := skv.kv[string(key)]
	if !ok {
		return ErrKeyNotFound
	}
	if string(value) != string(actual) {
		return ErrValueNotMatch
	}
	delete(skv.kv, string(key))
	return nil
}

func (skv *SimpleKV) Hash() (string, error) {
	h := crypto.SHA256.New()
	for key, value := range skv.kv {
		_, err := h.Write([]byte(key))
		if err != nil {
			return "", fmt.Errorf("hash error: %w", err)
		}
		_, err = h.Write(value)
		if err != nil {
			return "", fmt.Errorf("hash error: %w", err)
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
