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

func NewTransaction(nonce int, value int, from account.Address, to account.Address) Transaction {
	txn := Transaction{}
	txn.Nonce = nonce
	txn.Value = value
	txn.From = from
	txn.To = to
	txn.V = "abc"
	txn.S = "def"
	txn.R = "ghi"
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
	Nonce int
	From  account.Address
	To    account.Address
	Value int
	V     string
	R     string
	S     string
}

func (t *Transaction) String() string {
	return fmt.Sprintf("{nonce: %d, value: %d, from: %s, to: %s}", t.Nonce, t.Value, t.From.String()[:6]+"...",
		t.To.String()[:6]+"...")
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

func (t *SignedTransaction) HashBytes() []byte {
	return hash(t)
}

func (t *SignedTransaction) String() string {
	return fmt.Sprintf("{txn=%s, digest=%s, sig=%s}", t.Txn.String(),
		hex.EncodeToString(t.Digest)[:6]+"...", hex.EncodeToString(t.Signature)[:6]+"...")
}
