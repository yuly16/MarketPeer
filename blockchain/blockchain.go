package blockchain

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/blockchain/messaging"
	"go.dedis.ch/cs438/blockchain/miner"
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/blockchain/transaction"
	"go.dedis.ch/cs438/blockchain/wallet"
	"go.dedis.ch/cs438/logging"
	"go.dedis.ch/cs438/peer"
)

// FIXME: it might will expose to the outside world

type FullNodeConf struct {
	PeerMessager peer.Messager
	Messaging messaging.Messager
	Addr      string
	// TODO: let's not worry about security at this time
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
	Bootstrap  *block.BlockChain
	KVFactory  storage.KVFactory
	Account    *account.Account

	BlockTransactions int // how many transactions in a block
}

// FullNode is a Wallet as well as a Miner
type FullNode struct {
	logger      zerolog.Logger
	messager    messaging.Messager
	networkAddr string
	accountAddr *account.Address
	*wallet.Wallet
	*miner.Miner

}

// NewFullNode create a new full node, here we need to specify the transport layer
func NewFullNode(conf *FullNodeConf) *FullNode {
	m := miner.NewMiner(miner.MinerConf{
		Addr: conf.Addr, Messaging: conf.Messaging,
		Bootstrap: conf.Bootstrap, KVFactory: conf.KVFactory, AccountAddr: conf.Account.GetAddr()})

	w := wallet.NewWallet(wallet.WalletConf{
		Addr: conf.Addr, Messaging: conf.Messaging,
		PrivateKey: conf.PrivateKey, PublicKey: conf.PublicKey, KVFactory: conf.KVFactory, Account: conf.Account})

	f := &FullNode{Wallet: w, Miner: m, messager: conf.Messaging,
		networkAddr: conf.Addr, accountAddr: conf.Account.GetAddr()}
	f.logger = logging.RootLogger.With().Str("FullNode", fmt.Sprintf("%s", conf.Addr)).Logger()
	f.logger.Info().Msg("created")
	return f
}

func (f *FullNode) AddPeer(nodes ...*FullNode) {
	addrs := make([]string, 0, len(nodes))
	for _, node := range nodes {
		addrs = append(addrs, node.GetAddr())
	}
	f.messager.AddPeer(addrs...)
}

func (f *FullNode) GetAddr() string { return f.networkAddr }

func (f *FullNode) GetAccountAddr() *account.Address { return f.accountAddr }

func (f *FullNode) Start() {
	f.logger.Info().Msg("full node starting...")
	f.messager.Start()
	f.Miner.Start()
	f.Wallet.Start()
}

func (f *FullNode) Stop() {
	f.logger.Info().Msg("full node stopping...")
	f.messager.Stop()
	f.Miner.Stop()
	f.Wallet.Stop()
}

func (f *FullNode) GetChain() *block.BlockChain {
	return f.Miner.GetChain()
}

type BlockChain interface {
	// TODO: pointer or value?
	SubmitTxn(transaction.Transaction) error
	Verify(transaction.Transaction) error
}

// ETHz has Ether. So EPFl has Epfer
type Epfereum struct {
	BlockChain // TODO
}
