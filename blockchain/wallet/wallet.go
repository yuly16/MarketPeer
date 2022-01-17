package wallet

import (
	"crypto/rand"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"

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

	syncFutures     map[int]chan *types.SyncAccountReplyMessage                                       // TODO: dont consider evict now
	verifyFutures   map[transaction.SignedTransactionHandle]chan *types.VerifyTransactionReplyMessage // TODO: dont consider evict now
	verifyThreshold int
}

func NewWallet(conf WalletConf) *Wallet {
	w := Wallet{}
	w.messaging = conf.Messaging
	w.addr = conf.Addr
	w.publicKey = PublicKey{conf.PublicKey, crypto.FromECDSAPub(conf.PublicKey)}
	w.privateKey = PrivateKey{conf.PrivateKey, crypto.FromECDSA(conf.PrivateKey)}
	w.account = conf.Account
	w.syncFutures = make(map[int]chan *types.SyncAccountReplyMessage)
	w.verifyFutures = make(map[transaction.SignedTransactionHandle]chan *types.VerifyTransactionReplyMessage)
	w.verifyThreshold = 0
	w.logger = logging.RootLogger.With().Str("Wallet", fmt.Sprintf("%s", conf.Addr)).Logger()
	w.logger.Info().Msgf("wallet created:\n pubKey=%s, priKey=%s, account=%s",
		w.publicKey.String(), w.privateKey.String(), w.account.String())
	w.registerCallbacks()
	//sha256.New().Write([]byte(w.publicKey))
	return &w
}

func (w *Wallet) Start() {
	go w.syncAccountd()
}

func (w *Wallet) Stop() {}

func (w *Wallet) GetAccount() *account.Account {
	return w.account
}

// VerifyTxn verify if a transaction is valid
func (w *Wallet) VerifyTxn() error {
	return nil
}

func (w *Wallet) SyncAccount() error {
	// sync account state with neighbors
	stamp := int(time.Now().UnixMilli())

	syncMsg := &types.SyncAccountMessage{Timestamp: stamp, NetworkAddr: w.addr, Addr: *w.account.GetAddr()}
	neis := w.messaging.GetNeighbors()
	neis = append(neis, w.addr)
	w.logger.Info().Msgf("sync account with neis: %v", neis)
	future := make(chan *types.SyncAccountReplyMessage, len(neis)+1)
	w.syncFutures[stamp] = future
	waitfor := 0
	for _, nei := range neis {
		if err := w.messaging.Unicast(nei, syncMsg); err != nil {
			w.logger.Err(fmt.Errorf("sync account error: %w", err)).Send()
		}
		waitfor += 1
		w.logger.Debug().Msgf("syncMsg unicasted to %s", nei)
	}
	waitfor = int(math.Min(3, float64(waitfor)))
	majority := int(math.Ceil(float64(waitfor) / 2))
	states := make(map[string]*account.State) // state hash -> states
	stateCnts := make(map[string]int)         // state hash -> count

	w.logger.Info().Msgf("sync account started to waiting for reply")
	for i := 0; i < waitfor; i++ {
		reply := <-future
		newState := &reply.State
		newStateHash := newState.Hash()
		states[newStateHash] = newState
		if _, ok := stateCnts[newStateHash]; ok {
			stateCnts[newStateHash] += 1
		} else {
			stateCnts[newStateHash] = 1
		}
	}
	for k, v := range stateCnts {
		if v >= majority {
			w.logger.Info().Msgf("sync account, %d node agree on the state: %s", v, states[k])
			w.account.SetState(states[k])
			return nil
		}
	}
	w.logger.Warn().Msgf("sync account fail, received states: %v", states)
	return fmt.Errorf("sync account fail, received states: %v", states)
}

// TransferEpfer to dest
func (w *Wallet) TransferEpfer(dest account.Address, epfer int) error {
	// create a transaction and send
	// first do a SyncAccount?
	err := w.SyncAccount()
	for err != nil {
		w.logger.Warn().Msgf("value transfer sync account fail: %v", err)
		err = w.SyncAccount()
	}
	// first do a local balance check
	if w.account.GetBalance() < epfer {
		return fmt.Errorf("not enough balance, actual=%d, transfer=%d", w.account.GetBalance(), epfer)
	}
	// create the transaction
	txn := transaction.NewTransaction(w.account.GetNonce(), epfer, *w.GetAccount().GetAddr(), dest)
	// submit
	handle := w.SubmitTxn(txn)
	time.Sleep(1 * time.Second)
	// verify for a time
	begin := time.Now()
	err = w.VerifyTransaction(handle)
	for err != nil {
		err = w.VerifyTransaction(handle)
		time.Sleep(500 * time.Millisecond)
		if time.Since(begin) > 3*time.Second {
			return fmt.Errorf("transaction failed")
		}
	}
	return nil
}

func (w *Wallet) VerifyTransaction(handle *transaction.SignedTransactionHandle) error {
	verifyMsg := &types.VerifyTransactionMessage{w.addr, *handle}
	neis := w.messaging.GetNeighbors()
	neis = append(neis, w.addr)
	w.logger.Info().Msgf("verify txns with neis: %v", neis)
	future := make(chan *types.VerifyTransactionReplyMessage, len(neis)+1)
	w.verifyFutures[*handle] = future
	waitfor := 0
	for _, nei := range neis {
		if err := w.messaging.Unicast(nei, verifyMsg); err != nil {
			w.logger.Err(fmt.Errorf("verify txn error: %w", err)).Send()
		}
		waitfor += 1
		w.logger.Debug().Msgf("verifyTxnMsg unicasted to %s", nei)
	}
	waitfor = int(math.Min(3, float64(waitfor)))
	longestBlocksAfter := -1

	w.logger.Info().Msgf("sync account started to waiting for reply")
	for i := 0; i < waitfor; i++ {
		reply := <-future
		if reply.BlocksAfter > longestBlocksAfter {
			longestBlocksAfter = reply.BlocksAfter
		}
	}
	if longestBlocksAfter >= w.verifyThreshold {
		return nil
	}
	return fmt.Errorf("longestBlocksAfter = %d < verifyThreshold=%d", longestBlocksAfter, w.verifyThreshold)
}

// ShowAccount returns account to application
func (w *Wallet) ShowAccount() AccountInfo {
	// first sync the account
	err := w.SyncAccount()
	for err != nil {
		w.logger.Warn().Msgf("ShowAccount sync account fail: %v", err)
		err = w.SyncAccount()
	}
	s := make(map[string]interface{})
	w.account.GetState().StorageRoot.For(func(key string, value interface{}) error {
		s[key] = value
		return nil
	})
	return AccountInfo{Addr: w.account.GetAddr().String(), Balance: w.account.GetBalance(), Storage: s}
}

// Trigger contract sends empty transaction to the contract account, validated by account address
func (w *Wallet) TriggerContract(dest account.Address) error {
	err := w.SyncAccount()
	for err != nil {
		w.logger.Warn().Msgf("trigger contract sync account fail: %v", err)
		err = w.SyncAccount()
	}

	// create the transaction (value not care)
	txn := transaction.NewTransaction(w.account.GetNonce(), 0, *w.GetAccount().GetAddr(), dest)
	// submit
	handle := w.SubmitTxn(txn)
	time.Sleep(1 * time.Second)
	// verify for a time
	begin := time.Now()
	err = w.VerifyTransaction(handle)
	for err != nil {
		err = w.VerifyTransaction(handle)
		time.Sleep(500 * time.Millisecond)
		if time.Since(begin) > 3*time.Second {
			return fmt.Errorf("transaction failed")
		}
	}
	return nil
}

// Buyer propose a contract by submitting a special transaction
// transaction.Type = transaction.CREATE_CONTRACT
func (w *Wallet) ProposeContract() (string, error) {
	err := w.SyncAccount()
	for err != nil {
		w.logger.Warn().Msgf("trigger contract sync account fail: %v", err)
		err = w.SyncAccount()
	}

	// create propose contract transaction (with random address)
	bytesBegin := []byte{0, 0, 0, 0}
	bytesEnd := make([]byte, 4)
	_, err = rand.Read(bytesEnd)
	if err != nil {
		return "error", err
	}
	contract_address := append(bytesBegin, bytesEnd...)
	contract_addr_8b := [8]byte{}
	copy(contract_addr_8b[:], contract_address[len(contract_address)-8:])

	txn := transaction.NewProposeContractTransaction(w.account.GetNonce(), 0, *w.GetAccount().GetAddr(), *account.NewAddress(contract_addr_8b))
	// submit
	handle := w.SubmitTxn(txn)
	time.Sleep(1 * time.Second)
	// verify for a time
	begin := time.Now()
	err = w.VerifyTransaction(handle)
	for err != nil {
		err = w.VerifyTransaction(handle)
		time.Sleep(500 * time.Millisecond)
		if time.Since(begin) > 3*time.Second {
			return "error", fmt.Errorf("transaction failed")
		}
	}
	return string(contract_address), nil
}

// wallet can submit a transaction
// transaction is signed or not?
// how is digital coin represented?
func (w *Wallet) SubmitTxn(txn transaction.Transaction) *transaction.SignedTransactionHandle {
	signed := w.signTxn(txn)
	txnMessage := types.WalletTransactionMessage{Txn: signed}

	err := w.messaging.Broadcast(txnMessage)
	if err != nil {
		w.logger.Error().Msg("submitTxn: broadcast a transantion error. ")
	}
	w.logger.Info().Msgf(fmt.Sprintf("broadcast a txn:%s", txnMessage.Txn.String()))
	return &transaction.SignedTransactionHandle{signed.Hash()}
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
	w.messaging.RegisterMessageCallback(types.SyncAccountReplyMessage{}, w.SyncAccountReplyMessageCallback)
	w.messaging.RegisterMessageCallback(types.VerifyTransactionReplyMessage{}, w.VerifyTxnReplyMessageCallback)

}

//--------------The following code is just for debug -------------------//

func (w *Wallet) Test_submitTxn() {
	txn := transaction.NewTransaction(1, 2, *w.account.GetAddr(), *w.account.GetAddr())
	w.SubmitTxn(txn)
}
