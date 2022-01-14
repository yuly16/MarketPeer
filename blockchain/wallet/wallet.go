package wallet

import (
	"crypto/rsa"
	"fmt"
	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/messaging"
	"go.dedis.ch/cs438/blockchain/transaction"
	"go.dedis.ch/cs438/logging"
)

type WalletConf struct {
	Messaging  messaging.Messager
	Addr       string
	PrivateKey rsa.PrivateKey
	PublicKey  rsa.PublicKey
}

// Wallet can submit a txn
// a node is a wallet if it has a wallet
type Wallet struct {
	logger zerolog.Logger

	messaging messaging.Messager
	// TODO: wallet's view of Account is different from Epfer network view
	account account.Account
	addr    string

	publicKey  rsa.PublicKey
	privateKey rsa.PrivateKey
}

// TODO: is it like a factory mode?
func NewWallet(conf WalletConf) *Wallet {
	w := Wallet{}
	w.messaging = conf.Messaging
	w.addr = conf.Addr
	w.publicKey = conf.PublicKey
	w.privateKey = conf.PrivateKey
	w.logger = logging.RootLogger.With().Str("Wallet", fmt.Sprintf("%s", conf.Addr)).Logger()
	w.logger.Info().Msg("created")
	w.registerCallbacks()
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
