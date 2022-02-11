package conn

import (
	"crypto/rand"
	"fmt"
	"net"

	"github.com/Melenium2/kademlia"
	"github.com/Melenium2/kademlia/internal/table/node"
)

type UDPConn interface {
	ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error)
	WriteToUDP(b []byte, addr *net.UDPAddr) (n int, err error)
	Close() error
	LocalAddr() net.Addr
}

type rpc struct {
	requestID []byte
	self      *node.Node
	request   Packet
	resCh     chan Packet
	errCh     chan error
}

func newRpc(self *node.Node, request Packet) *rpc {
	r := rpc{
		requestID: make([]byte, 8),
		self:      self,
		request:   request,
		resCh:     make(chan Packet, 1),
		errCh:     make(chan error, 1),
	}

	_, _ = rand.Read(r.requestID)

	return &r
}

type Transport struct {
	// established udp connection.
	conn UDPConn

	// queue of messages we need to send.
	callQueue map[kademlia.ID][]*rpc
	// map with messages we already send.
	pendingCalls map[kademlia.ID]*rpc

	nextCallCh   chan *rpc
	cancelCallCh chan *rpc
}

func NewListener(conn UDPConn) *Transport {
	return &Transport{
		conn:         conn,
		callQueue:    make(map[kademlia.ID][]*rpc),
		pendingCalls: make(map[kademlia.ID]*rpc),
		nextCallCh:   make(chan *rpc, 10), // todo change here to constant
		cancelCallCh: make(chan *rpc, 10),
	}
}

func (t *Transport) Loop() error {
	for {
		select {
		case nextCall := <-t.nextCallCh:
			_ = nextCall
			// where we need register new call in call queue
			// and add call to pending calls.

		case canceledCall := <-t.cancelCallCh:
			_ = canceledCall
			// where we need remove call from call queue
			// and remove it from pending call.
		}
	}
}

func (t *Transport) SendPing(node *node.Node) (*Pong, error) {
	// set something to ping message
	req := &Ping{}
	remoteCall := t.call(node, req)
	defer t.pruneCall(remoteCall)

	packet, err := consume(remoteCall.resCh, remoteCall.errCh)
	if err != nil {
		return nil, err
	}

	pong, ok := packet.(*Pong)
	if !ok {
		return nil, fmt.Errorf("%w to ping request", ErrWrongMessageType)
	}

	return pong, nil
}

func (t *Transport) SendPong() error {
	return nil
}

func (t *Transport) call(node *node.Node, req Packet) *rpc {
	remoteCall := newRpc(node, req)

	t.nextCallCh <- remoteCall

	return remoteCall
}

// pruneCall clears all response channels from rpc and remove rpc call
// from pending calls.
func (t *Transport) pruneCall(rpc *rpc) {
	for {
		select {
		case <-rpc.resCh:
		case <-rpc.errCh:
		case t.cancelCallCh <- rpc:
			return
		}
	}
}

func (t *Transport) nextCall() {
	// unmarshal and send to some ip
}

func consume(packetCh chan Packet, errCh chan error) (Packet, error) {
	var (
		packet Packet
		err    error
	)

	select {
	case packet = <-packetCh:
	case err = <-errCh:
	}

	return packet, err
}
