package miner

import (
	"fmt"
	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/blockchain/messaging"
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/logging"
)

// miner Conf
type MinerConf struct {
	Messaging messaging.Messager
	Addr      string
	Bootstrap storage.BlockChain
}

// Miner is a full node in Epfer network
type Miner struct {
	logger zerolog.Logger

	messaging messaging.Messager
	addr      string

	chain storage.BlockChain
}

func NewMiner(conf MinerConf) *Miner {
	m := Miner{}
	m.messaging = conf.Messaging
	m.addr = conf.Addr
	m.chain = conf.Bootstrap
	m.logger = logging.RootLogger.With().Str("Miner", fmt.Sprintf("%s", conf.Addr)).Logger()
	m.logger.Info().Msg("created")
	m.registerCallbacks()
	return &m
}

// precisely speaking, miner dont submit txns, they verify txns.
// txns are broadcasted by other nodes or wallets to the Epfer network.
// if this node is also a miner, then it will verify the txn sent by itself first by default.
// a node is a miner if it has a miner. then the miner will register verify-related callbacks
// func (m *Miner) submitTxn() {}

func (m *Miner) Start() {}

func (m *Miner) Stop() {}

func (m *Miner) verifyTxn() {}

func (m *Miner) submitBlock() {}

func (m *Miner) registerCallbacks() {}

// daemons
func (m *Miner) verifyTxnd() {}

func (m *Miner) verifyBlockd() {}

// callbacks
