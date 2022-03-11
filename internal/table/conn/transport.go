package conn

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net"
	"time"

	"github.com/Melenium2/kademlia"
	"github.com/Melenium2/kademlia/internal/table/kbuckets"
	"github.com/Melenium2/kademlia/internal/table/node"
	"github.com/Melenium2/kademlia/pkg/logger"
)

const (
	Timeout                = 1 * time.Second
	MaxMessageSize         = 1000
	TotalNodesPacketsLimit = 5
	TotalNodeSendLimit     = 16
)

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

type UDPConn interface {
	ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error)
	WriteToUDP(b []byte, addr *net.UDPAddr) (n int, err error)
}

type KBuckets interface {
	WhoAmI() *node.Node
	BucketAtDistance(dist int) *kbuckets.Bucket
}

// Transport is structure for providing access to UDP network.
// Transport provide various API for communicate with another
// nodes in network.
type Transport struct {
	// Established udp connection.
	conn  UDPConn
	store KBuckets
	log   logger.Logger

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
func NewTransport(conn UDPConn, store KBuckets) *Transport {
	return &Transport{
		conn:         conn,
		store:        store,
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
	go t.readFromNetwork(ctx)

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

// send convert provided packet to raw bytes and send by the UDP connection to provided
// address.
func (t *Transport) send(fromID kademlia.ID, req Packet, addr *net.UDPAddr) error {
	body := Marshal(fromID.Bytes(), req)

	_, err := t.conn.WriteToUDP(body, addr)
	if err != nil {
		t.log.Warnf("can not send body: \n%s to udp socket %s", hex.Dump(body), addr.String())

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

	packet, err := t.consume(remoteCall.resCh, remoteCall.errCh)
	if err != nil {
		return nil, err
	}

	pong, ok := packet.(*Pong)
	if !ok {
		// this case should be never executed.
		return nil, fmt.Errorf("%w to ping request", ErrWrongMessageType)
	}

	return pong, nil
}

// consume wait for first message from packetCh or errCh and return
// that comes first.
func (t *Transport) consume(packetCh chan Packet, errCh chan error) (Packet, error) {
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

// FindNode make call for provided node with provided 'distances', this node should return
// list of nodes which distance contains in 'distances' slice.
func (t *Transport) FindNode(node *node.Node, distances []uint) ([]*node.Node, error) {
	var (
		reqID = GenerateReqID()
		req   = &FindNodes{ReqID: reqID, Distances: distances}
	)

	remoteCall := t.call(reqID, node, req)
	defer t.pruneCall(remoteCall)

	return t.consumeNodes(remoteCall, distances)
}

// consumeNodes consumes all incoming NodesList in provided range if 'distances', if some node ID
// not in 'distance' slice, than we remove it from result slice.
//
// This function returns nodes, with unique ID's.
func (t *Transport) consumeNodes(call *rpc, distances []uint) ([]*node.Node, error) {
	var (
		nodes           []*node.Node
		seen            = make(map[kademlia.ID]struct{})
		err             error
		received, total = 0, -1
	)

	for {
		select {
		case packet := <-call.resCh:
			packetNodes, ok := packet.(*NodesList)
			if !ok {
				return nil, fmt.Errorf("can not consume messages, reason %w", ErrWrongMessageType)
			}

			if total == -1 {
				total = min(int(packetNodes.Count), TotalNodesPacketsLimit)
			}

			for _, nextNode := range packetNodes.Nodes {
				id := nextNode.ID()

				if err = t.validateNode(call.self.ID(), id, distances); err != nil {
					t.log.Warnf("node with ID %d is invalid, reason %s", id.Bytes(), err)

					continue
				}

				if _, ok = seen[id]; ok {
					t.log.Warnf("got duplicate record with ID %s, IP %s", hex.EncodeToString(id.Bytes()), nextNode.IP())

					continue
				}

				seen[id] = struct{}{}

				nodes = append(nodes, nextNode)
			}

			if received++; received == total {
				return nodes, nil
			}
		case err = <-call.errCh:
			return nodes, err
		}
	}
}

// validateNode finds node.LogDistance between provided kademlia.ID's and checks
// if distance exists in provided argument, if not that means that result was
// incorrect.
func (t *Transport) validateNode(self, incoming kademlia.ID, distance []uint) error {
	// todo check that node is valid connection target. We need to check IP and Port.
	logDistance := node.LogDistance(self, incoming)
	if !containsUint(uint(logDistance), distance) {
		return ErrNotMatchAnyDistance
	}

	return nil
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

// readFromNetwork starts reading cycle from UDP network, and handles
// each incoming frame. Max size of incoming frame is equals to MaxMessageSize.
//
// readFromNetwork is blocking thread. For closing this function you must
// cancel the provided context.
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
				t.log.Errorf("UDP read error, closing for read network cycle, reason %s", err)

				return
			}

			t.handleNetworkPacket(buf[:n], addr)
		}
	}
}

// handleNetworkPacket handles each incoming packet.
//
// If function can not unmarshal raw bytes to the packet, the error will be logged.
// Another case where error will be logged, if type of packet unknown.
// nolint:cyclop
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
	case *FindNodes:
		if err = t.handleFindNode(id, p, addr); err != nil {
			t.log.Error(err.Error())

			return
		}
	case *NodesList:
		if err = t.handleNodeList(id, p, addr); err != nil {
			t.log.Error(err.Error())

			return
		}
	default:
		t.log.Warnf("got unknown packet type %+v", packet)
	}
}

// handlePing message and response with pong message to the sender.
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

// handlePong message and validate it. If message is valid, then trying to complete rpc call.
func (t *Transport) handlePong(id []byte, pong *Pong, addr *net.UDPAddr) error {
	return t.handleIncomingResponse(PongMessage, id, pong, addr)
}

// handleFindNode message and finds in local bucket storage nodes with provided distances, then
// form group and send result back to requester.
func (t *Transport) handleFindNode(id []byte, p *FindNodes, addr *net.UDPAddr) error {
	nodes := t.findNodesByDistance(p.Distances)

	if len(nodes) == 0 {
		if err := t.send(kademlia.NewIDFromSlice(id), &NodesList{ReqID: p.ReqID, Count: 1}, addr); err != nil {
			return fmt.Errorf("can not send back NodesList, reason %w", err)
		}
	}

	groups := t.packNodesByGroups(p.ReqID, nodes)

	for _, group := range groups {
		if err := t.send(kademlia.NewIDFromSlice(id), group, addr); err != nil {
			return fmt.Errorf("can not send back NodesList, reason %w", err)
		}
	}

	return nil
}

// findNodesByDistance check nodes with provided distances inside local bucket store.
// If we find nodes more than TotalNodeSendLimit then cut it and return.
func (t *Transport) findNodesByDistance(distances []uint) []*node.Node {
	var (
		processed = make(map[uint]struct{}, len(distances))
		nodes     []*node.Node
	)

	for _, distance := range distances {
		_, seen := processed[distance]
		if seen || distance > 256 {
			continue
		}

		if distance == 0 {
			nodes = append(nodes, t.store.WhoAmI())
		} else {
			nodesAtDistance := t.store.BucketAtDistance(int(distance))
			nodes = append(nodes, nodesAtDistance.Entries...)
		}

		processed[distance] = struct{}{}

		if len(nodes) >= TotalNodeSendLimit {
			return nodes[:TotalNodeSendLimit]
		}
	}

	return nodes
}

// packNodesByGroups groups all provided nodes to group up to TotalNodesPacketsLimit.
//
// For example if 16 nodes will be provided, then this function will separate it to
// groups with 5 - 5 - 5 - 1 nodes.
func (t *Transport) packNodesByGroups(id []byte, nodes []*node.Node) []*NodesList {
	var (
		packCount = math.Ceil(float64(len(nodes)) / TotalNodesPacketsLimit)
		nodeLists = make([]*NodesList, int(packCount))
	)

	for i := range nodeLists {
		nodeLists[i] = &NodesList{
			ReqID: id,
			Count: uint8(packCount),
			Nodes: make([]*node.Node, 0, TotalNodesPacketsLimit),
		}
	}

	for i, n := range nodes {
		currIndex := int(math.Floor(float64(i) / TotalNodesPacketsLimit))

		nodeLists[currIndex].Nodes = append(nodeLists[currIndex].Nodes, n)
	}

	return nodeLists
}

// handleNodeList message. If message is valid trying to cancel rpc call.
func (t *Transport) handleNodeList(id []byte, p *NodesList, addr *net.UDPAddr) error { // nolint:interfacer
	return t.handleIncomingResponse(NodesListMessage, id, p, addr)
}

// handleIncomingResponse, common function for each incoming responses to requests. Function will
// find necessary rpc call and validate it with incoming Packet, if it is valid then provide
// packet to result channel of rpc call.
func (t *Transport) handleIncomingResponse(respType byte, id []byte, p Packet, addr *net.UDPAddr) error {
	kadeID := kademlia.NewIDFromSlice(id)

	pc, ok := t.pendingCalls[kadeID]
	if !ok {
		return fmt.Errorf("node with arrived ID not found, got wrong ID")
	}

	selfAddr := &net.UDPAddr{
		IP:   pc.self.IP(),
		Port: pc.self.UDPPort(),
	}

	if err := t.validateIncomingPacket(respType, pc.request, p, selfAddr, addr); err != nil {
		return err
	}

	pc.ApplyTimeout(Timeout)

	pc.resCh <- p

	return nil
}

// validateIncomingPacket compare RequestID, Type, IP and Port of two packets, if all right then return nothing.
func (t *Transport) validateIncomingPacket(
	waitFor byte, pending, incoming Packet, pendingAddr, incomingAddr *net.UDPAddr,
) error {
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

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func containsUint(target uint, ar []uint) bool {
	for i := 0; i < len(ar); i++ {
		if ar[i] == target {
			return true
		}
	}

	return false
}
