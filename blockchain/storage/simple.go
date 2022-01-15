package storage

import (
	"crypto"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type SimpleKV struct {
	kv map[string]interface{}
}

func NewSimpleKV() *SimpleKV {
	return &SimpleKV{kv: make(map[string]interface{})}
}

func (skv *SimpleKV) Get(key string) (interface{}, error) {
	value, ok := skv.kv[key]
	if !ok {
		return nil, ErrKeyNotFound
	}
	return value, nil
}

func (skv *SimpleKV) Put(key string, value interface{}) error {
	skv.kv[key] = value
	return nil
}

func (skv *SimpleKV) Del(key string) error {
	_, ok := skv.kv[key]
	if !ok {
		return ErrKeyNotFound
	}

	delete(skv.kv, key)
	return nil
}

func (skv *SimpleKV) Copy() KV {
	serialized, err := json.Marshal(skv)
	if err != nil {
		panic(err)
	}
	var ret *SimpleKV = &SimpleKV{}
	err = json.Unmarshal(serialized, ret)
	if err != nil {
		panic(err)
	}
	return ret
}

func (skv *SimpleKV) String() string {
	ret := "{"
	for key, value := range skv.kv {
		ret += fmt.Sprintf("%s->%s", key, value)
		ret += ","
	}
	return ret + "}"
}

func (skv *SimpleKV) Hash() string {
	h := crypto.SHA256.New()
	for key, value := range skv.kv {
		_, err := h.Write([]byte(key))
		if err != nil {
			panic(err)
		}

		bytes, err := json.Marshal(value)
		if err != nil {
			panic(err)
		}
		_, err = h.Write(bytes)
		if err != nil {
			panic(err)
		}
	}

	return hex.EncodeToString(h.Sum(nil))
}
