package storage

import (
	"crypto"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

type SimpleKV struct {
	Internal map[string]interface{}
}

func NewSimpleKV() *SimpleKV {
	return &SimpleKV{Internal: make(map[string]interface{})}
}

func (skv *SimpleKV) For(compute func(key string, value interface{}) error) error {
	for k, v := range skv.Internal {
		err := compute(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (skv *SimpleKV) Get(key string) (interface{}, error) {
	value, ok := skv.Internal[key]
	if !ok {
		return nil, ErrKeyNotFound
	}
	return value, nil
}

func (skv *SimpleKV) Put(key string, value interface{}) error {
	skv.Internal[key] = value
	return nil
}

func (skv *SimpleKV) Del(key string) error {
	_, ok := skv.Internal[key]
	if !ok {
		return ErrKeyNotFound
	}

	delete(skv.Internal, key)
	return nil
}

// keys is not efficient, FIXME
func (skv *SimpleKV) sortedKeys() []string {
	ret := make([]string, 0, len(skv.Internal))
	for k := range skv.Internal {
		ret = append(ret, k)
	}
	sort.Strings(ret)
	return ret
}

// TODO: serialize and deserialize, it really fucked!
func (skv *SimpleKV) Copy() KV {
	ret := CreateSimpleKV()
	for k, v := range skv.Internal {
		switch vv := v.(type) {
		case Copyable:
			ret.Put(k, vv.Copy())
		default:
			ret.Put(k, v)
		}
	}
	return ret
	//serialized, err := json.Marshal(skv)
	//if err != nil {
	//	panic(err)
	//}
	//var ret *SimpleKV = &SimpleKV{map[string]interface{}{}}
	//err = json.Unmarshal(serialized, ret)
	//for key, value := range ret.Internal {
	//	vv, ok := value.(map[string]interface{})
	//	if !ok {
	//		panic(value)
	//	}
	//	state := account.NewStateBuilder(CreateSimpleKV).
	//		SetNonce(vv["Nonce"].(uint)).
	//		SetBalance(vv["Balance"].(uint)).
	//		SetCode(vv["Code"].(string))
	//	storageRoot := vv["StorageRoot"].(map[string]interface{})
	//	for srk, srv := range storageRoot {
	//		state.SetKV(srk, srv)
	//	}
	//	ret.Put(key, state.Build())
	//}
	//if err != nil {
	//	panic(err)
	//}
	//return ret
}

func (skv *SimpleKV) String() string {
	ret := "{"
	for _, key := range skv.sortedKeys() {
		value, ok := skv.Internal[key]
		if !ok {
			continue
		}
		ret += fmt.Sprintf("%s->%s", key, value)
		ret += ","
	}

	return ret + "}"
}

func (skv *SimpleKV) Hash() string {
	h := crypto.SHA256.New()
	for _, key := range skv.sortedKeys() {
		value, ok := skv.Internal[key]
		if !ok {
			continue
		}
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
