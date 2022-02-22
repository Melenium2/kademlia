package conn

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/Melenium2/kademlia"
	"github.com/Melenium2/kademlia/internal/table/node"
	"github.com/Melenium2/kademlia/pkg/logger"
)

const (
	Timeout        = 1 * time.Second
	MaxMessageSize = 1000
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
	timeout   *time.Timer
	request   Packet
	resCh     chan Packet
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
	if r.timeout != nil {
		return r.timeout.Stop()
	}

	return false
}

// Close response channels.
func (r *rpc) Close() {
	close(r.resCh)
	close(r.errCh)
}

func newRPC(reqID []byte, self *node.Node, request Packet) *rpc {
	r := rpc{
		requestID: reqID,
		self:      self,
		request:   request,
		resCh:     make(chan Packet, 1),
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

	// Messages queue we need to sendNext.
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
		// todo change here to constant
		cancelCallCh: make(chan *rpc, 10), // nolint:gomnd
		nextCallCh:   make(chan *rpc, 10), // nolint:gomnd
	}
}

// Loop is main request/response cycle here. Loop make queue incoming requests
// and handle out coming responses.
//
// If you needed to close cycle, you need juts cancel provided context.
func (t *Transport) Loop(ctx context.Context) error {
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
		case <-ctx.Done():
			return context.Canceled
		}
	}
}

// nextPending handles next item in rpc calls queue by provided ID.
// If no items in queue with this ID or some call already in pending state,
// function just return without any message.
func (t *Transport) nextPending(id kademlia.ID) {
	queue := t.callQueue[id]
	if len(queue) == 0 || t.pendingCalls[id] != nil {
		return
	}

	call := queue[0]
	t.pendingCalls[id] = call
	_ = t.sendNext(call)

	if len(queue) == 1 {
		delete(t.callQueue, id)
	} else {
		queue = queue[1:]
		t.callQueue[id] = queue
	}
}

// removeFromPending removes last rpc call with provided ID from pending
// state.
//
// If this function called with ID, which not contains inside
// pending mapping, then function will call FATAL exit, because
// this situation impossible if all components will work in normal
// mode.
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

// sendNext rpc message to client.
//
// This function, also, apply timeout to the rpc call. If it not
// completes in Timeout time, then rpc call return error.
func (t *Transport) sendNext(call *rpc) error {
	addr := &net.UDPAddr{
		IP:   call.self.IP(),
		Port: call.self.UDPPort(),
	}

	if err := t.send(call.self.ID(), call.request, addr); err != nil {
		return err
	}

	call.ApplyTimeout(Timeout)

	return nil
}

func (t *Transport) send(fromID kademlia.ID, req Packet, addr *net.UDPAddr) error {
	body := Marshal(fromID.Bytes(), req)

	_, err := t.conn.WriteToUDP(body, addr)
	if err != nil {
		t.log.Warnf("can not sendNext body: %s, to udp socket %s", body, addr.String())

		return err
	}

	return nil
}

// SendPing message to provided node.
func (t *Transport) SendPing(node *node.Node) (*Pong, error) {
	var (
		reqID = GenerateReqID()
		req   = &Ping{ReqID: reqID}
	)

	remoteCall := t.call(reqID, node, req)
	defer t.pruneCall(remoteCall)

	packet, err := consume(remoteCall.resCh, remoteCall.errCh)
	if err != nil {
		return nil, err
	}

	// TODO dont forrget to cacnel timeout

	pong, ok := packet.(*Pong)
	if !ok {
		// this case should be never executed.
		return nil, fmt.Errorf("%w to ping request", ErrWrongMessageType)
	}

	return pong, nil
}

// call creates new rpc message and pass it to queue.
func (t *Transport) call(reqID []byte, node *node.Node, req Packet) *rpc {
	remoteCall := newRPC(reqID, node, req)

	t.nextCallCh <- remoteCall

	return remoteCall
}

// pruneCall clears all rpc response channels and remove call
// from pending calls.
func (t *Transport) pruneCall(rpc *rpc) {
	rpc.Close()

	for range rpc.resCh {
	}

	for range rpc.errCh {
	}

	t.cancelCallCh <- rpc
}

// nolint:unused
func (t *Transport) readFromNetwork(ctx context.Context) {
	buf := make([]byte, MaxMessageSize)

	for {
		select {
		case <-ctx.Done():
			t.log.Warnf("read network cycle is closed, reason %w", ctx.Err())

			return
		default:
			n, addr, err := t.conn.ReadFromUDP(buf)
			if err != nil {
				t.log.Errorf("UDP read error, closing for read network cycle, reason %w", err)

				return
			}

			t.handleNetworkPacket(buf[:n], addr)
		}
	}
}

// nolint:unused
func (t *Transport) handleNetworkPacket(body []byte, addr *net.UDPAddr) {
	packet, id, err := Unmarshal(body)
	if err != nil {
		t.log.Errorf("can not unmarshal incoming message, reason %w", err)
	}

	switch p := packet.(type) {
	case *Ping:
		if err = t.handlePing(id, p, addr); err != nil {
			t.log.Error(err.Error())

			return
		}
	case *Pong:
		if err = t.handlePong(id, p, addr); err != nil {
			t.log.Error(err.Error())

			return
		}
	}

	t.log.Warnf("got unknown packet type %+v", packet)
}

// nolint:unused
func (t *Transport) handlePing(id []byte, ping *Ping, addr *net.UDPAddr) error {
	ip := addr.IP.To4()

	pong := &Pong{
		ReqID: ping.ReqID,
		IP:    ip,
		Port:  uint16(addr.Port),
	}

	if err := t.send(kademlia.NewIDFromSlice(id), pong, addr); err != nil {
		return fmt.Errorf("can not send pong back to ping request, request ID %s, reason %w", ping.ReqID, err)
	}

	return nil
}

func (t *Transport) handlePong(id []byte, pong *Pong, addr *net.UDPAddr) error {
	kadeID := kademlia.NewIDFromSlice(id)

	pc, ok := t.pendingCalls[kadeID]
	if !ok {
		return fmt.Errorf("node with arrived ID not found, got wrong ID")
	}

	selfAddr := &net.UDPAddr{
		IP:   pc.self.IP(),
		Port: pc.self.UDPPort(),
	}

	if err := validateIncomingPacket(PongMessage, pc.request, pong, selfAddr, addr); err != nil {
		return err
	}

	pc.ApplyTimeout(Timeout)

	pc.resCh <- pong

	return nil
}

func validateIncomingPacket(waitFor byte, pending, incoming Packet, pendingAddr, incomingAddr *net.UDPAddr) error {
	if !bytes.Equal(pending.GetRequestID(), incoming.GetRequestID()) {
		return fmt.Errorf(
			"%w, request ID of found node (%s) not equals to incoming request ID (%s)",
			ErrValidate,
			pending.GetRequestID(),
			incoming.GetRequestID(),
		)
	}

	if waitFor != incoming.IAm() {
		return fmt.Errorf(
			"%w, got unexpected response type, expected = %X, got = %X",
			ErrValidate,
			waitFor,
			incoming.IAm(),
		)
	}

	if !pendingAddr.IP.Equal(incomingAddr.IP) {
		return fmt.Errorf(
			"%w, got incorrect IP address, expected = %s, got = %s",
			ErrValidate,
			pendingAddr.IP,
			incomingAddr.IP,
		)
	}

	if pendingAddr.Port != incomingAddr.Port {
		return fmt.Errorf(
			"%w, got incorrent PORT, expected = %d, got = %d",
			ErrValidate,
			pendingAddr.Port,
			incomingAddr.Port,
		)
	}

	return nil
}

// consume wait for first message from packetCh or errCh and return
// that comes first.
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
