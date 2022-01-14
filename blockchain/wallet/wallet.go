package wallet

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/messaging"
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/blockchain/transaction"
	"go.dedis.ch/cs438/logging"
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

// TODO: use WalletBuilder to give more fine-grained control
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

// transferEpfer to dest
func (w *Wallet) transferEpfer(dest account.Account, epfer int) {

}

// wallet can submit a transaction
// transaction is signed or not?
// how is digital coin represented?
func (w *Wallet) submitTxn(txn transaction.Transaction) {

}

func (w *Wallet) signTxn() {}

func (w *Wallet) registerCallbacks() {}
