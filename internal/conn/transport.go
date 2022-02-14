package conn

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/Melenium2/kademlia"
	"github.com/Melenium2/kademlia/internal/table/node"
	"github.com/Melenium2/kademlia/pkg/logger"
)

const Timeout = 1 * time.Second

type UDPConn interface {
	ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error)
	WriteToUDP(b []byte, addr *net.UDPAddr) (n int, err error)
	Close() error
	LocalAddr() net.Addr
}

type rpc struct {
	requestID []byte
	self      *node.Node
	timeout   *time.Timer
	request   Packet
	resCh     chan []byte
	errCh     chan error
}

func (r *rpc) ApplyTimeout(timeout time.Duration) {
	if r.timeout != nil {
		r.timeout.Stop()
	}

	var (
		timer *time.Timer
		done  = make(chan struct{})
	)

	time.AfterFunc(timeout, func() {
		<-done
		r.errCh <- errors.New("got rpc timeout")
	})

	r.timeout = timer
	close(done)
}

func (r *rpc) StopTimeout() bool {
	return r.timeout.Stop()
}

func newRpc(reqID []byte, self *node.Node, request Packet) *rpc {
	r := rpc{
		requestID: reqID,
		self:      self,
		request:   request,
		resCh:     make(chan []byte, 1),
		errCh:     make(chan error, 1),
	}

	return &r
}

// Transport is structure for providing access to UDP network.
// Transport provide various API for communicate with another
// nodes in network.
type Transport struct {
	// Established udp connection.
	conn UDPConn
	log  logger.Logger

	// Messages queue we need to send.
	callQueue map[kademlia.ID][]*rpc
	// Map with messages already sent.
	pendingCalls map[kademlia.ID]*rpc

	// Channel which provided access to next rpc call.
	nextCallCh chan *rpc
	// Channel which provide access to already
	// returned rpc requests.
	cancelCallCh chan *rpc
}

// NewTransport create instance of Transport.
func NewTransport(conn UDPConn) *Transport {
	return &Transport{
		conn:         conn,
		log:          logger.GetLogger(),
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
			id := nextCall.self.ID()
			t.callQueue[id] = append(t.callQueue[id], nextCall)
			t.nextPending(id)
			// where we need register new call in call queue
			// and add call to pending calls.

		case canceledCall := <-t.cancelCallCh:
			id := canceledCall.self.ID()
			t.removeFromPending(id)
			t.nextPending(id)
			// where we need remove call from call queue
			// and remove it from pending call.
		}
	}
}

func (t *Transport) nextPending(id kademlia.ID) {
	queue := t.callQueue[id]
	if len(queue) == 0 || t.pendingCalls[id] != nil {
		return
	}

	call := queue[0]
	t.pendingCalls[id] = call
	_ = t.send(call)

	if len(queue) == 1 {
		delete(t.callQueue, id)
	} else {
		queue = queue[1:]
		t.callQueue[id] = queue
	}
}

func (t *Transport) removeFromPending(id kademlia.ID) {
	var (
		call *rpc
		ok   bool
	)

	if call, ok = t.pendingCalls[id]; !ok {
		t.log.Fatal("trying to remove inactive call, this is unreal")

		return
	}

	_ = call.StopTimeout()
	delete(t.pendingCalls, id)
}

func (t *Transport) send(call *rpc) error {
	addr := &net.UDPAddr{
		IP:   call.self.IP(),
		Port: call.self.UDPPort(),
	}

	body := Marshal(call.request)

	_, err := t.conn.WriteToUDP(body, addr)
	if err != nil {
		t.log.Warnf("can not send body: %s, to udp socket %s", body, addr.String())

		return err
	}

	call.ApplyTimeout(Timeout)

	return nil
}

func (t *Transport) SendPing(node *node.Node) (*Pong, error) {
	var (
		reqID = GenerateReqID()
		req   = &Ping{ReqID: reqID}
	)

	remoteCall := t.call(reqID, node, req)
	defer t.pruneCall(remoteCall)

	packet, err := consume(PongMessage, remoteCall.resCh, remoteCall.errCh)
	if err != nil {
		return nil, err
	}

	pong, ok := packet.(*Pong)
	if !ok {
		return nil, fmt.Errorf("%w to ping request", ErrWrongMessageType)
	}

	return pong, nil
}

func (t *Transport) call(reqID []byte, node *node.Node, req Packet) *rpc {
	remoteCall := newRpc(reqID, node, req)

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

func consume(bodytype byte, packetCh chan []byte, errCh chan error) (Packet, error) {
	var (
		packet    Packet
		rawPacket []byte
		err       error
	)

	select {
	case rawPacket = <-packetCh:
		packet, err = Unmarshal(bodytype, rawPacket)
		if err != nil {
			return nil, err
		}
	case err = <-errCh:
	}

	return packet, err
}
