package conn

import (
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
	self    *node.Node
	waitFor byte
}

type Transport struct {
	// established udp connection.
	conn UDPConn

	// queue of messages we need to send.
	callQueue map[kademlia.ID][]*rpc
	// map with messages we already send.
	activeCalls map[kademlia.ID]*rpc
}

func NewListener(conn UDPConn) *Transport {
	return &Transport{
		conn:        conn,
		callQueue:   make(map[kademlia.ID][]*rpc),
		activeCalls: make(map[kademlia.ID]*rpc),
	}
}

func (t *Transport) Loop() error {
	return nil
}

func (t *Transport) SendPing(node *node.Node) error {
	req := &Ping{}
	resp := t.call(node, PongMessage, req)
	_ = resp

	// ------
	// we need to send call and write new rpc to all maps
	// then we need to create handler of all possible responses
	// and wait response for ping message.
	// ------

	// im see two ways:
	// 1) we should do calls one by one, then we can control flow
	// 	  of messages and not use mutex.
	// 2) we can send messages concurrently, but we need map it to requests
	//    when response arrived, and here we need to use mutex for concurrent
	//    access to map with requests.
	return nil
}

func (t *Transport) SendPong() error {
	return nil
}

func (t *Transport) call(node *node.Node, responseType byte, req Packet) *rpc {

	return nil
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
