package wallet

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/messaging"
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/blockchain/transaction"
	"go.dedis.ch/cs438/logging"
	"go.dedis.ch/cs438/types"
)

type WalletConf struct {
	Messaging  messaging.Messager
	Addr       string // sock address
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
	KVFactory  storage.KVFactory
	Account    *account.Account
}

type PrivateKey struct {
	*ecdsa.PrivateKey
	bytes []byte
}

func (pri *PrivateKey) String() string {
	return hex.EncodeToString(pri.bytes)[:8] + "..."
}

type PublicKey struct {
	*ecdsa.PublicKey
	bytes []byte
}

func (pub *PublicKey) String() string {
	return hex.EncodeToString(pub.bytes)[:8] + "..."
}

// Wallet can submit a txn
// a node is a wallet if it has a wallet
type Wallet struct {
	logger zerolog.Logger

	messaging messaging.Messager
	// TODO: wallet's view of Account is different from Epfer network view
	account *account.Account
	addr    string

	publicKey  PublicKey
	privateKey PrivateKey
}

func NewWallet(conf WalletConf) *Wallet {
	w := Wallet{}
	w.messaging = conf.Messaging
	w.addr = conf.Addr
	w.publicKey = PublicKey{conf.PublicKey, crypto.FromECDSAPub(conf.PublicKey)}
	w.privateKey = PrivateKey{conf.PrivateKey, crypto.FromECDSA(conf.PrivateKey)}
	w.account = conf.Account

	w.logger = logging.RootLogger.With().Str("Wallet", fmt.Sprintf("%s", conf.Addr)).Logger()
	w.logger.Info().Msgf("wallet created:\n pubKey=%s, priKey=%s, account=%s",
		w.publicKey.String(), w.privateKey.String(), w.account.String())
	w.registerCallbacks()
	//sha256.New().Write([]byte(w.publicKey))
	return &w
}

func (w *Wallet) Start() {}

func (w *Wallet) Stop() {}

// TransferEpfer to dest
func (w *Wallet) TransferEpfer(dest account.Account, epfer int) {

}

// wallet can submit a transaction
// transaction is signed or not?
// how is digital coin represented?
func (w *Wallet) SubmitTxn(txn transaction.Transaction) {
	txnMessage := types.WalletTransactionMessage{Txn: w.signTxn(txn)}

	err := w.messaging.Broadcast(txnMessage)
	if err != nil {
		w.logger.Error().Msg("submitTxn: broadcast a transantion error. ")
	}
	w.logger.Info().Msgf(fmt.Sprintf("broadcast a txn:%s", txnMessage.Txn.String()))
}

func (w *Wallet) hash(data interface{}) []byte {
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

func (w *Wallet) signTxn(txn transaction.Transaction) transaction.SignedTransaction {
	signedTxn, err := transaction.NewSignedTransaction(txn, w.privateKey.PrivateKey)
	if err != nil {
		w.logger.Error().Msgf("sign transaction error! ")
	}
	return signedTxn
	//signature, err := crypto.Sign(w.hash(txn), w.privateKey.PrivateKey)
	//publicKey, err := crypto.Ecrecover(w.hash(txn), signature)
	//fmt.Println(w.publicKey.bytes)
	//fmt.Println(publicKey)
	//ok := crypto.VerifySignature(w.publicKey.bytes, w.hash(txn), signature[:len(signature)-1])
	////ok := ecdsa.Verify(w.publicKey.PublicKey, w.hash(txn), r, s)
	//if err != nil {
	//	w.logger.Error().Msgf("sign transaction fails. ")
	//}
	//fmt.Println(ok)
}

func (w *Wallet) registerCallbacks() {
	//w.messaging.RegisterMessageCallback(types.WalletTransactionMessage{}, w.WalletTxnMsgCallback)

}

//--------------The following code is just for debug -------------------//

func (w *Wallet) Test_submitTxn() {
	txn := transaction.NewTransaction(1, 2, *w.account.GetAddr(), *w.account.GetAddr())
	w.SubmitTxn(txn)
}
