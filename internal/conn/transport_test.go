package conn

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/Melenium2/kademlia"
	"github.com/Melenium2/kademlia/internal/conn/mocks"
	"github.com/Melenium2/kademlia/internal/table/node"
	"github.com/Melenium2/kademlia/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	addr = &net.UDPAddr{
		IP:   net.IPv4(1, 1, 1, 1),
		Port: 5222,
	}
	expectedPong = &Pong{
		ReqID: []byte("13123123"),
		IP:    addr.IP,
		Port:  uint16(addr.Port),
	}
)

func TestConsume_Should_consume_new_packet_from_result_channel(t *testing.T) {
	var (
		packetCh = make(chan Packet, 1)
		errCh    = make(chan error, 1)
	)

	go func() {
		time.Sleep(300 * time.Millisecond)
		packetCh <- expectedPong
	}()

	packet, err := consume(packetCh, errCh)
	assert.NoError(t, err)

	pong, ok := packet.(*Pong)
	assert.True(t, ok)
	assert.Equal(t, expectedPong, pong)
}

func TestConsume_Should_got_error_while_waiting_for_new_packet(t *testing.T) {
	var (
		packetCh = make(chan Packet, 1)
		errCh    = make(chan error, 1)
	)

	go func() {
		time.Sleep(100 * time.Millisecond)
		errCh <- io.ErrClosedPipe
	}()

	_, err := consume(packetCh, errCh)
	assert.Error(t, err)
	assert.Equal(t, io.ErrClosedPipe, err)
}

func TestTransport_PruneCall_Should_retrieve_all_items_from_rpc_channels_and_then_send_rpc_to_cancel_channel(t *testing.T) {
	rpcCall := &rpc{
		resCh: make(chan Packet, 1),
		errCh: make(chan error, 1),
	}

	rpcCall.resCh <- expectedPong
	rpcCall.errCh <- io.ErrClosedPipe

	transport := Transport{
		cancelCallCh: make(chan *rpc, 1),
	}

	transport.pruneCall(rpcCall)
	assert.Len(t, rpcCall.resCh, 0)
	assert.Len(t, rpcCall.errCh, 0)
	assert.Len(t, transport.cancelCallCh, 1)
	// checking, that channels are closed.
	_, ok := <-rpcCall.errCh
	assert.False(t, ok)
	_, ok = <-rpcCall.resCh
	assert.False(t, ok)
}

var (
	testNode = &node.Node{
		Node: kademlia.NewNode(addr),
	}
)

func TestTransport_SendPing_Should_send_ping_request_and_got_pong_response(t *testing.T) {
	transport := Transport{
		nextCallCh:   make(chan *rpc, 1),
		cancelCallCh: make(chan *rpc, 1),
	}

	go func() {
		time.Sleep(300 * time.Millisecond)

		rpcCall := <-transport.nextCallCh
		rpcCall.resCh <- expectedPong
	}()

	pong, err := transport.SendPing(testNode)
	assert.NoError(t, err)
	assert.Equal(t, expectedPong, pong)
	assert.Len(t, transport.cancelCallCh, 1)
}

var (
	testCall = &rpc{
		requestID: nil,
		self: &node.Node{
			Node: kademlia.NewNode(addr),
		},
		request: expectedPong,
	}
	id       = testCall.self.ID()
	fakeConn = func() UDPConn {
		testbody := Marshal(id.Bytes(), testCall.request)

		fake := mocks.UDPConn{}
		fake.
			On("WriteToUDP", testbody, addr).
			Return(0, nil)

		return &fake
	}
)

func TestTransport_Send_Should_marshal_and_send_rpc_request_to_udp_connection_and_apply_request_timeout(t *testing.T) {
	transport := Transport{
		conn: fakeConn(),
	}

	err := transport.sendNext(testCall)
	assert.NoError(t, err)
}

func TestTransport_Send_Should_return_error_if_can_not_send_body_to_udp_conn(t *testing.T) {
	body := Marshal(id[:], testCall.request)

	fakeConn := mocks.UDPConn{}
	fakeConn.
		On("WriteToUDP", body, addr).
		Return(0, io.ErrClosedPipe)

	transport := Transport{
		conn: &fakeConn,
		log:  logger.GetLogger(),
	}

	err := transport.sendNext(testCall)
	assert.Error(t, err)
	assert.Equal(t, io.ErrClosedPipe, err)
}

func TestTransport_RemoveFromPending_Should_remove_rpc_call_with_provided_id_from_local_state(t *testing.T) {
	transport := NewTransport(nil)
	transport.pendingCalls[id] = testCall

	transport.removeFromPending(id)

	assert.Len(t, transport.pendingCalls, 0)
}

func TestTransport_NextPending_Should_send_next_pending_call_and_remove_it_from_queue(t *testing.T) {
	transport := NewTransport(fakeConn())

	transport.callQueue[id] = append(transport.callQueue[id], testCall)

	transport.nextPending(id)

	assert.Len(t, transport.pendingCalls, 1)
	assert.Equal(t, testCall, transport.pendingCalls[id])
	assert.Len(t, transport.callQueue, 0)
}

func TestTransport_NextPending_Should_send_next_pending_call_but_in_queue_should_stay_one_more(t *testing.T) {
	transport := NewTransport(fakeConn())

	transport.callQueue[id] = append(transport.callQueue[id], testCall, testCall)

	transport.nextPending(id)

	assert.Len(t, transport.pendingCalls, 1)
	assert.Equal(t, testCall, transport.pendingCalls[id])
	assert.Len(t, transport.callQueue, 1)
}

func TestTransport_NextPending_Should_return_from_func_if_call_queue_is_empty(t *testing.T) {
	transport := NewTransport(fakeConn())

	transport.nextPending(id)

	assert.Len(t, transport.pendingCalls, 0)
	assert.Len(t, transport.callQueue, 0)
}

func TestTransport_NextPending_Should_return_from_func_if_queue_by_provided_id_is_empty(t *testing.T) {
	anotherID, _ := kademlia.GenerateID()

	transport := NewTransport(fakeConn())
	transport.callQueue[anotherID] = append(transport.callQueue[anotherID], testCall)

	transport.nextPending(id)

	assert.Len(t, transport.pendingCalls, 0)
	assert.Len(t, transport.callQueue, 1)
	assert.Len(t, transport.callQueue[anotherID], 1)
}

func TestTransport_NextPending_Should_return_from_second_func_because_rpc_call_with_provided_id_already_in_pending_state(t *testing.T) {
	transport := NewTransport(fakeConn())

	transport.callQueue[id] = append(transport.callQueue[id], testCall)

	transport.nextPending(id)
	transport.nextPending(id)

	assert.Len(t, transport.pendingCalls, 1)
	assert.Equal(t, testCall, transport.pendingCalls[id])
	assert.Len(t, transport.callQueue, 0)
}

func TestTransport_Loop_Should_receive_new_call_and_add_it_to_call_queue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	transport := NewTransport(fakeConn())

	go func() {
		err := transport.Loop(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	}()

	time.Sleep(200 * time.Millisecond)

	transport.nextCallCh <- testCall

	time.Sleep(200 * time.Millisecond)

	// we close loop here for safe data access. If not call close here
	// we can get datarace.
	cancel()
	// wait for goroutine closing
	time.Sleep(100 * time.Millisecond)

	// nextPending func should remove first item from queue and set it
	// to pendingCalls
	assert.Len(t, transport.callQueue, 0)
	assert.Len(t, transport.callQueue[id], 0)
	assert.Len(t, transport.pendingCalls, 1)
	assert.Equal(t, testCall, transport.pendingCalls[id])
}

func TestTransport_Loop_Should_receive_cancel_call_message_and_remove_last_pending_call_from_store(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	transport := NewTransport(fakeConn())
	transport.pendingCalls[id] = testCall

	go func() {
		err := transport.Loop(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	}()

	time.Sleep(200 * time.Millisecond)

	transport.cancelCallCh <- testCall

	time.Sleep(200 * time.Millisecond)

	// we close loop here for safe data access. If not call close here
	// we can get datarace.
	cancel()

	time.Sleep(100 * time.Millisecond)

	assert.Len(t, transport.pendingCalls, 0)
	assert.Nil(t, transport.pendingCalls[id])

}

func TestTransport_Loop_Should_close_read_cycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	transport := NewTransport(fakeConn())

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := transport.Loop(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestValidateIncomingPacket(t *testing.T) {
	var tt = []struct {
		name     string
		waitFor  byte
		pending  Packet
		incoming Packet
		pendAddr *net.UDPAddr
		incAddr  *net.UDPAddr
		expected error
	}{
		{
			name:    "should successfully validate incoming packet",
			waitFor: PongMessage,
			pending: &Ping{
				ReqID: []byte("13123123"),
			},
			incoming: expectedPong,
			pendAddr: &net.UDPAddr{
				IP:   expectedPong.IP,
				Port: int(expectedPong.Port),
			},
			incAddr: &net.UDPAddr{
				IP:   net.IPv4(1, 1, 1, 1),
				Port: 5222,
			},
			expected: nil,
		},
		{
			name: "should return error if request IDs is not same",
			pending: &Ping{
				ReqID: []byte("111111111"),
			},
			incoming: &Pong{
				ReqID: expectedPong.ReqID,
			},
			expected: ErrValidate,
		},
		{
			name:    "should return error if type of incoming message it not same with expected",
			waitFor: PingMessage,
			pending: &Ping{
				ReqID: expectedPong.ReqID,
			},
			incoming: &Pong{
				ReqID: expectedPong.ReqID,
			},
			expected: ErrValidate,
		},
		{
			name:    "should return error if IP address in incoming message is not same with pending node IP",
			waitFor: PongMessage,
			pending: &Ping{
				ReqID: expectedPong.ReqID,
			},
			incoming: &Pong{
				ReqID: expectedPong.ReqID,
			},
			pendAddr: &net.UDPAddr{
				IP: net.IPv4(2, 2, 2, 2),
			},
			incAddr: &net.UDPAddr{
				IP: expectedPong.IP,
			},
			expected: ErrValidate,
		},
		{
			name:    "should return error if PORT field in incoming message is not same with pending node PORT",
			waitFor: PongMessage,
			pending: &Ping{
				ReqID: expectedPong.ReqID,
			},
			incoming: &Pong{
				ReqID: expectedPong.ReqID,
			},
			pendAddr: &net.UDPAddr{
				IP:   expectedPong.IP,
				Port: 1111,
			},
			incAddr: &net.UDPAddr{
				IP:   expectedPong.IP,
				Port: int(expectedPong.Port),
			},
			expected: ErrValidate,
		},
	}

	t.Parallel()

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateIncomingPacket(tc.waitFor, tc.pending, tc.incoming, tc.pendAddr, tc.incAddr)
			assert.ErrorIs(t, err, tc.expected)
		})
	}
}

var kadeID = kademlia.ID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}

func TestTransport_HandlePong_Should_find_pending_call_and_send_incoming_packet_res_channel(t *testing.T) {
	rpcCall := &rpc{
		requestID: []byte("13123123"),
		self:      testNode,
		request: &Ping{
			ReqID: []byte("13123123"),
		},
		resCh: make(chan Packet, 1),
	}

	transport := Transport{
		pendingCalls: make(map[kademlia.ID]*rpc),
	}
	transport.pendingCalls[kadeID] = rpcCall

	err := transport.handlePong(kadeID.Bytes(), expectedPong, addr)
	require.NoError(t, err)

	packet := <-rpcCall.resCh
	assert.Equal(t, expectedPong, packet.(*Pong))
}

func TestTransport_HandlePong_Should_cancel_call_with_error_if_time_is_out(t *testing.T) {
	rpcCall := &rpc{
		requestID: []byte("13123123"),
		self:      testNode,
		request: &Ping{
			ReqID: []byte("13123123"),
		},
		resCh: make(chan Packet, 1),
		errCh: make(chan error, 1),
	}

	transport := Transport{
		pendingCalls: make(map[kademlia.ID]*rpc),
	}
	transport.pendingCalls[kadeID] = rpcCall

	err := transport.handlePong(kadeID.Bytes(), expectedPong, addr)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 1200)

	err = <-rpcCall.errCh
	assert.Error(t, err)
}

func TestTransport_HandlePong_Should_return_error_if_can_node_find_rpc_call_by_id(t *testing.T) {
	newID := kademlia.ID{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}

	rpcCall := &rpc{
		requestID: []byte("13123123"),
		self:      testNode,
		request: &Ping{
			ReqID: []byte("13123123"),
		},
		resCh: make(chan Packet, 1),
		errCh: make(chan error, 1),
	}

	transport := Transport{
		pendingCalls: make(map[kademlia.ID]*rpc),
	}
	transport.pendingCalls[kadeID] = rpcCall

	err := transport.handlePong(newID.Bytes(), expectedPong, addr)
	assert.Error(t, err)
}

func TestTransport_HandlePong_Should_return_error_if_can_not_validate_incoming_packet(t *testing.T) {
	rpcCall := &rpc{
		requestID: []byte("13123123"),
		self:      testNode,
		request: &Ping{
			ReqID: []byte("wrong request id"),
		},
		resCh: make(chan Packet, 1),
		errCh: make(chan error, 1),
	}

	transport := Transport{
		pendingCalls: make(map[kademlia.ID]*rpc),
	}
	transport.pendingCalls[kadeID] = rpcCall

	err := transport.handlePong(kadeID.Bytes(), expectedPong, addr)
	assert.Error(t, err)
}

func TestTransport_HandlePing_Should_send_response_with_pong_body(t *testing.T) {
	var (
		ping         = &Ping{ReqID: expectedPong.ReqID}
		expectedBody = Marshal(kadeID.Bytes(), expectedPong)
	)

	internalFakeConn := &mocks.UDPConn{}
	internalFakeConn.
		On("WriteToUDP", expectedBody, addr).
		Return(len(expectedBody), nil).
		Once()

	transport := Transport{
		conn: internalFakeConn,
	}

	err := transport.handlePing(kadeID.Bytes(), ping, addr)
	assert.NoError(t, err)
}

func TestTransport_HandlePing_Should_return_error_if_can_not_write_to_connection(t *testing.T) {
	var (
		ping         = &Ping{ReqID: expectedPong.ReqID}
		expectedBody = Marshal(kadeID.Bytes(), expectedPong)
	)

	internalFakeConn := &mocks.UDPConn{}
	internalFakeConn.
		On("WriteToUDP", expectedBody, addr).
		Return(0, io.ErrClosedPipe).
		Once()

	transport := Transport{
		log:  logger.GetLogger(),
		conn: internalFakeConn,
	}

	err := transport.handlePing(kadeID.Bytes(), ping, addr)
	assert.Error(t, err)
}

func TestTransport_HandleNetworkPacket_Should_unmarshal_ping_and_handle_it_without_error(t *testing.T) {
	var (
		ping = &Ping{
			ReqID: expectedPong.ReqID,
		}
		incomingBody = Marshal(id.Bytes(), ping)
	)

	transport := Transport{
		conn: fakeConn(),
	}

	transport.handleNetworkPacket(incomingBody, addr)
}

func TestTransport_HandleNetworkPacket_Should_return_error_from_handling_ping_function_if_can_not_send_response_back(t *testing.T) {
	var (
		ping = &Ping{
			ReqID: expectedPong.ReqID,
		}
		incomingBody = Marshal(id.Bytes(), ping)
		expectedBody = Marshal(id.Bytes(), expectedPong)
	)

	internalFakeConn := &mocks.UDPConn{}
	internalFakeConn.
		On("WriteToUDP", expectedBody, addr).
		Return(0, io.ErrClosedPipe).
		Once()

	transport := Transport{
		log:  logger.GetLogger(),
		conn: internalFakeConn,
	}

	transport.handleNetworkPacket(incomingBody, addr)
}

func TestTransport_HandleNetworkPacket_Should_unmarshal_pong_message_and_handle_it_then_send_message_to_result_channel_of_rpc_call(t *testing.T) {
	var (
		rpcCall = &rpc{
			requestID: []byte("13123123"),
			self:      testNode,
			request: &Ping{
				ReqID: []byte("13123123"),
			},
			resCh: make(chan Packet, 1),
		}
		incomingBody = Marshal(kadeID.Bytes(), expectedPong)
	)

	transport := Transport{
		pendingCalls: make(map[kademlia.ID]*rpc),
	}
	transport.pendingCalls[kadeID] = rpcCall

	transport.handleNetworkPacket(incomingBody, addr)
}

func TestTransport_HandleNetworkPacket_Should_return_error_while_processing_packet_because_this_rpc_call_was_unexpected(t *testing.T) {
	var (
		incomingBody = Marshal(kadeID.Bytes(), expectedPong)
	)

	transport := Transport{
		log:          logger.GetLogger(),
		pendingCalls: make(map[kademlia.ID]*rpc),
	}

	transport.handleNetworkPacket(incomingBody, addr)
}

var (
	nodeAddr = &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 11503,
	}
	reqAddr = &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 11504,
		Zone: "",
	}
	testping = &Ping{
		ReqID: expectedPong.ReqID,
	}
	testpong = &Pong{
		ReqID: expectedPong.ReqID,
		IP:    reqAddr.IP,
		Port:  uint16(reqAddr.Port),
	}
	incomingBody = Marshal(id.Bytes(), testping)
	resultPong   = Marshal(id.Bytes(), testpong)
)

func TestTransport_ReadFromNetwork_Should_read_and_process_ping_packet_then_close_with_context(t *testing.T) {
	buf := make([]byte, 1024)

	// start listening port.
	conn, err := net.ListenUDP("udp", nodeAddr)
	require.NoError(t, err)

	defer conn.Close()

	ctx, cancelFunc := context.WithCancel(context.Background())
	transport := Transport{
		log:  logger.GetLogger(),
		conn: conn,
	}

	// run read cycle.
	go transport.readFromNetwork(ctx)

	// create new udp "client".
	udpConn, err := net.DialUDP("udp", reqAddr, nodeAddr)
	require.NoError(t, err)

	defer udpConn.Close()

	// send ping to node.
	_, err = udpConn.Write(incomingBody)
	require.NoError(t, err)

	n, err := udpConn.Read(buf)
	require.NoError(t, err)
	require.Equal(t, resultPong, buf[:n])

	cancelFunc()
}
