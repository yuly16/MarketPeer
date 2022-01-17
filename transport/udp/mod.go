package udp

import (
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"go.dedis.ch/cs438/internal/traffic"
	"go.dedis.ch/cs438/transport"
)

// var _logger zerolog.Logger = zerolog.New(
// 	zerolog.NewConsoleWriter(
// 		func(w *zerolog.ConsoleWriter) { w.Out = os.Stderr },
// 		func(w *zerolog.ConsoleWriter) { w.TimeFormat = "15:04:05.000" })).
// 	With().Str("mod", "UDPSock").Timestamp().Logger()

const bufSize = 9200 // macos max udp datagram is 9216 bytes
const recvBufSize = 1000

// NewUDP returns a new udp transport implementation.
func NewUDP() transport.Transport {
	return &UDP{traffic: traffic.NewTraffic()}
}

// UDP implements a transport layer using UDP
//
// - implements transport.Transport
type UDP struct {
	traffic *traffic.Traffic
}

// CreateSocket implements transport.Transport
func (n *UDP) CreateSocket(address string) (transport.ClosableSocket, error) {
	conn, err := net.ListenPacket("udp", address)
	if err != nil {
		return nil, fmt.Errorf("cannot create udp socket: %w", err)
	}
	return &Socket{PacketConn: conn, recvBuf: make(chan readResult, recvBufSize),
		ins:  packets{data: make([]transport.Packet, 0, 100)},
		outs: packets{data: make([]transport.Packet, 0, 100)},
		traf: n.traffic}, nil
}

// Socket implements a network socket using UDP.
//
// - implements transport.Socket
// - implements transport.ClosableSocket
// NOTE: Socket shall not modify the packet
// NOTE: pay attention to the packet modify. From my perspective, socket shall
// 		 not hold the states of packets. It shall be held in the caller
type Socket struct {
	net.PacketConn
	ins     packets
	outs    packets
	recvBuf chan readResult
	traf    *traffic.Traffic
}

// wrap recv result
type readResult struct {
	pkt transport.Packet
	err error
}

// Close implements transport.Socket. It returns an error if already closed.
func (s *Socket) Close() error {
	return s.PacketConn.Close()
}

// Send implements transport.Socket
// timeout=0 means no timeout
func (s *Socket) Send(dest string, pkt transport.Packet, timeout time.Duration) error {
	pktBytes, err := pkt.Marshal()
	if err != nil {
		return fmt.Errorf("UDP send error: %w", err)
	}
	addr, err := net.ResolveUDPAddr("udp", dest)
	if err != nil {
		return fmt.Errorf("UDP send error: %w", err)
	}
	// prevent the func to block forever
	res := make(chan error, 1)
	go func() {
		_, err := s.WriteTo(pktBytes, addr)
		res <- err
	}()
	// no timeout
	if timeout.Milliseconds() == 0 {
		err := <-res
		if err != nil {
			return fmt.Errorf("UDP send error: %w", err)
		}
	} else {
		select {
		case err := <-res:
			if err != nil {
				return fmt.Errorf("UDP send error: %w", err)
			}
		case <-time.After(timeout):
			return fmt.Errorf("UDP send error: %w", transport.TimeoutErr(timeout))
		}
	}

	// success send FIXME: there might be inconsistency, since we dont add error send
	// to outs. But the packet might be received by others(timeout for example)
	// here we dont need to copy, since getOuts do the copy, we have the full control
	s.outs.add(pkt)
	s.traf.LogSent(pkt.Header.RelayedBy, dest, pkt)
	return nil
}

// Recv implements transport.Socket. It blocks until a packet is received, or
// the timeout is reached. In the case the timeout is reached, return a
// TimeoutErr.
// it is a producer as well as a consumer of recvBuf
func (s *Socket) Recv(timeout time.Duration) (transport.Packet, error) {
	// try to consume a buffered recv result
	select {
	case recvRes := <-s.recvBuf:
		if recvRes.err != nil {
			return transport.Packet{}, fmt.Errorf("UDP Recv error: %w", recvRes.err)
		} else {
			pkt := recvRes.pkt
			s.ins.add(pkt)
			s.traf.LogRecv(pkt.Header.RelayedBy, s.GetAddress(), pkt.Copy()) // FIXME: this copy might be avoided
			return pkt.Copy(), nil
		}
	default:
	}

	// start to produce a result
	go func() {
		var pkt transport.Packet
		var err error
		buf := make([]byte, bufSize)
		n, _, errRead := s.ReadFrom(buf)
		// first process n bytes then process error, as indicated by the
		// `ReadFrom` interface. FIXME: if errParse=nil but errRead!=nil, shall we bypass read error?
		errParse := pkt.Unmarshal(buf[:n])
		if errRead == nil {
			err = errParse
		} else if errParse == nil {
			err = errRead
		} else {
			err = fmt.Errorf("unmarshal error(%v) plus read error: %w", errParse, errRead)
		}
		s.recvBuf <- readResult{pkt, err}
	}()

	// block call
	if timeout.Milliseconds() == 0 {
		timeout = math.MaxInt32 * time.Second
	}
	// try to receive a result within timeout, mostly likely the one it produces but it could also
	// be results produced by other `Recv` call. But it doesn't matter since #produce=#consume eventually
	select {
	case recvRes := <-s.recvBuf:
		if recvRes.err != nil {
			return transport.Packet{}, fmt.Errorf("UDP Recv error: %w", recvRes.err)
		} else {
			// success recv from the caller semantics
			// NOTE: here the inconsistency might be resolved. Since every `add` corresponding to
			// 		 a success recv, and there is no ignored recv with `recvTimeoutBuf`
			pkt := recvRes.pkt
			s.ins.add(pkt)
			s.traf.LogRecv(pkt.Header.RelayedBy, s.GetAddress(), pkt.Copy()) // FIXME: this copy might be avoided
			return recvRes.pkt.Copy(), nil
		}
	case <-time.After(timeout):
		return transport.Packet{}, transport.TimeoutErr(timeout)
	}
}

// GetAddress implements transport.Socket. It returns the address assigned. Can
// be useful in the case one provided a :0 address, which makes the system use a
// random free port.
func (s *Socket) GetAddress() string {
	return s.LocalAddr().String()
}

// GetIns implements transport.Socket
func (s *Socket) GetIns() []transport.Packet {
	return s.ins.getAll()
}

// GetOuts implements transport.Socket
func (s *Socket) GetOuts() []transport.Packet {
	return s.outs.getAll()
}

// utility class
type packets struct {
	sync.Mutex
	data []transport.Packet
}

func (p *packets) add(pkt transport.Packet) {
	p.Lock()
	defer p.Unlock()

	p.data = append(p.data, pkt)
}

func (p *packets) getAll() []transport.Packet {
	p.Lock()
	defer p.Unlock()

	res := make([]transport.Packet, len(p.data))

	for i, pkt := range p.data {
		res[i] = pkt.Copy()
	}

	return res
}
