package miner

import (
	"fmt"
	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/blockchain/messaging"
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/blockchain/transaction"
	"go.dedis.ch/cs438/logging"
	"go.dedis.ch/cs438/types"
	"sync/atomic"
)

// miner state
const (
	KILL = iota
	ALIVE
)

// miner Conf
type MinerConf struct {
	Messaging         messaging.Messager
	Addr              string
	Bootstrap         *block.BlockChain
	BlockTransactions int               // how many transactions in a block
	KVFactory         storage.KVFactory // kv factory to create Blocks
}

// Miner is a full node in Epfer network
type Miner struct {
	logger zerolog.Logger

	messaging messaging.Messager
	addr      string

	chain *block.BlockChain

	txnCh     chan *transaction.SignedTransaction
	blocktxns int               // how many transactions in a block
	kvFactory storage.KVFactory // kv factory to create Blocks

	// Service
	stat int32
}

func NewMiner(conf MinerConf) *Miner {
	m := Miner{}
	m.messaging = conf.Messaging
	m.addr = conf.Addr
	m.chain = conf.Bootstrap
	m.txnCh = make(chan *transaction.SignedTransaction, 100)
	m.blocktxns = conf.BlockTransactions
	m.kvFactory = conf.KVFactory
	m.logger = logging.RootLogger.With().Str("Miner", fmt.Sprintf("%s", conf.Addr)).Logger()
	m.logger.Info().Msgf("miner created:\n %s", m.chain.String())
	m.registerCallbacks()
	return &m
}

// precisely speaking, miner dont submit txns, they verify txns.
// txns are broadcasted by other nodes or wallets to the Epfer network.
// if this node is also a miner, then it will verify the txn sent by itself first by default.
// a node is a miner if it has a miner. then the miner will register verify-related callbacks
// func (m *Miner) submitTxn() {}

func (m *Miner) Start() {
	m.stat = ALIVE
	go m.verifyTxnd()
	go m.verifyBlockd()
}

func (m *Miner) Stop() {
	atomic.StoreInt32(&m.stat, KILL)
}

func (m *Miner) isKilled() bool {
	return atomic.LoadInt32(&m.stat) == KILL
}

func (m *Miner) submitBlock() {}

func (m *Miner) registerCallbacks() {
	m.messaging.RegisterMessageCallback(types.BlockMessage{}, m.BlockMsgCallback)
}


// ---------------------------------the code is just for testing------------------------

func (m *Miner) BroadcastBlock(block block.Block) {
	err := m.messaging.Broadcast(types.BlockMessage{Block: block})
	if err != nil {
		return
	}
}
