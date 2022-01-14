package messaging

import (
	"fmt"
	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/logging"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/registry"
	"go.dedis.ch/cs438/types"
)

// Messager corresponds to a communication ending point(ip:port)
// it offers
//   send functionalities: Unicast, Broadcast
//   route functionalities: AddPeer, SetRoutingEntry, GetRoutingTable
//   process functionalities: RegisterMessageCallback
type Messager interface {
	peer.Service
	RegisterMessageCallback(types.Message, registry.Exec)
	Unicast(dest string, msg types.Message) error
	Broadcast(msg types.Message) error
	AddPeer(addr ...string)
	GetRoutingTable() peer.RoutingTable
	SetRoutingEntry(origin, relayAddr string)
}

// RegistryMessager implements Messager
type RegistryMessager struct {
	logger      zerolog.Logger
	messager    peer.Messager
	msgRegistry registry.Registry
}

func NewRegistryMessager(addr string, messaging peer.Messager, registry registry.Registry) *RegistryMessager {
	m := &RegistryMessager{}
	m.logger = logging.RootLogger.With().Str("Messaging", fmt.Sprintf("%s", addr)).Logger()
	m.logger.Info().Msg("created")
	m.messager = messaging
	m.msgRegistry = registry
	return m
}

func (m *RegistryMessager) Start() error {
	m.messager.Start()
	return nil
}

func (m *RegistryMessager) Stop() error {
	m.messager.Stop()
	return nil
}

func (m *RegistryMessager) RegisterMessageCallback(message types.Message, exec registry.Exec) {
	m.msgRegistry.RegisterMessageCallback(message, exec)
}

func (m *RegistryMessager) Unicast(dest string, msg types.Message) error {
	err := m.doUnicast(dest, msg)
	if err != nil {
		return fmt.Errorf("unicast error: %w", err)
	}
	return nil
}

func (m *RegistryMessager) Broadcast(msg types.Message) error {
	err := m.doBroadcast(msg)
	if err != nil {
		return fmt.Errorf("unicast error: %w", err)
	}
	return nil
}

func (m *RegistryMessager) AddPeer(addr ...string) {
	m.messager.AddPeer(addr...)
}

func (m *RegistryMessager) GetRoutingTable() peer.RoutingTable {
	return m.messager.GetRoutingTable()
}

func (m *RegistryMessager) SetRoutingEntry(origin, relayAddr string) {
	m.messager.SetRoutingEntry(origin, relayAddr)
}

func (m *RegistryMessager) doBroadcast(msg types.Message) error {
	// first marshal the msg
	message, err := m.msgRegistry.MarshalMessage(msg)
	if err != nil {
		return err
	}
	err = m.messager.Broadcast(message)
	if err != nil {
		return err
	}
	return nil
}

func (m *RegistryMessager) doUnicast(dest string, msg types.Message) error {
	// first marshal the msg
	message, err := m.msgRegistry.MarshalMessage(msg)
	if err != nil {
		return err
	}
	err = m.messager.Unicast(dest, message)
	if err != nil {
		return err
	}
	return nil
}
