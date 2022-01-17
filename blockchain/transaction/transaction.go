package transaction

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"go.dedis.ch/cs438/blockchain/account"
)

const (
	EXEC_TXN = iota
	SET_STORAGE
	EXEC_CONTRACT
	CREATE_CONTRACT
)

var txnTypes = map[int]string{EXEC_TXN: "EXEC_TXN", SET_STORAGE: "SET_STORAGE", EXEC_CONTRACT: "EXEC_CONTRACT", CREATE_CONTRACT: "CREATE_CONTRACT"}

func NewTransaction(nonce int, value int, from account.Address, to account.Address) Transaction {
	txn := Transaction{}
	txn.Nonce = nonce
	txn.Value = value
	txn.From = from
	txn.Type = EXEC_TXN
	txn.To = to
	txn.V = "abc"
	txn.S = "def"
	txn.R = "ghi"

	txn.StoreV = ""
	return txn
}

func NewTriggerContractTransaction(nonce int, value int, from account.Address, to account.Address) Transaction {
	txn := Transaction{}
	txn.Nonce = nonce
	txn.Value = value
	txn.From = from
	txn.Type = EXEC_CONTRACT
	txn.To = to
	txn.V = "abc"
	txn.S = "def"
	txn.R = "ghi"

	txn.StoreV = ""
	return txn
}

func NewProposeContractTransaction(nonce int, value int, from account.Address, to account.Address, code string) Transaction {
	txn := Transaction{}
	txn.Nonce = nonce
	txn.Value = value
	txn.Type = CREATE_CONTRACT
	txn.From = from
	txn.To = to
	txn.Code = code
	txn.V = "abc"
	txn.S = "def"
	txn.R = "ghi"
	txn.StoreV = ""

	return txn
}

func NewSetStorageTransaction(nonce int, value int, from account.Address, key string, kvalue interface{}) Transaction {
	txn := Transaction{}
	txn.Nonce = nonce
	txn.Value = value
	txn.Type = SET_STORAGE
	txn.From = from
	txn.To = from
	txn.StoreK = key
	txn.StoreV = kvalue
	return txn
}

func hash(data interface{}) []byte {
	h := sha256.New()
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	if _, err := h.Write(bytes); err != nil {
		return nil
	}
	val := h.Sum(nil)
	return val
}

func NewSignedTransaction(txn Transaction, prv *ecdsa.PrivateKey) (SignedTransaction, error) {
	signedTxn := SignedTransaction{}
	signedTxn.Txn = txn
	signedTxn.Digest = hash(txn)
	signature, err := crypto.Sign(signedTxn.Digest, prv)
	if err != nil {
		return SignedTransaction{}, err

	}
	signedTxn.Signature = signature
	return signedTxn, nil
}

type Transaction struct {
	Nonce  int
	From   account.Address
	To     account.Address
	Type   int
	Code   string
	Value  int
	StoreK string
	StoreV interface{}
	V      string
	R      string
	S      string
}

func (t *Transaction) String() string {
	return fmt.Sprintf("{nonce: %d, value: %d, from: %s, to: %s, type: %s}", t.Nonce, t.Value, t.From.String()[:6]+"...",
		t.To.String()[:6]+"...", txnTypes[t.Type])
}

func (t *Transaction) Print() {
	fmt.Printf("transaction info: nonce: %d, value: %d, v: %s, r: %s, s:%s\n",
		t.Nonce, t.Value, t.V, t.R, t.S)
}

type SignedTransaction struct {
	Txn       Transaction
	Digest    []byte
	Signature []byte
}

// SignedTransactionHandle used as a handle to find the txn in the chain
type SignedTransactionHandle struct {
	Hash string // hash of the signed transaction
}

func (st *SignedTransactionHandle) String() string {
	return st.Hash
}

func (t *SignedTransaction) Hash() string {
	return hex.EncodeToString(t.HashBytes())
}

func (t *SignedTransaction) HashBytes() []byte {
	return hash(t)
}

func (t *SignedTransaction) String() string {
	return fmt.Sprintf("{txn=%s, digest=%s, sig=%s}", t.Txn.String(),
		hex.EncodeToString(t.Digest)[:6]+"...", hex.EncodeToString(t.Signature)[:6]+"...")
}
