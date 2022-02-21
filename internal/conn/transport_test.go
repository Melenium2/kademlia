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
)

var (
	expectedPong = &Pong{
		ReqID: []byte("13123123"),
		IP:    net.IPv4(1, 1, 1, 1),
		Port:  5222,
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
		Node: kademlia.NewNode(),
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
			Node: kademlia.NewNode(),
		},
		request: expectedPong,
	}
	id       = testCall.self.ID()
	fakeConn = func() UDPConn {
		testbody := Marshal(id[:], testCall.request)

		fake := mocks.UDPConn{}
		fake.
			On("WriteToUDP", testbody, &net.UDPAddr{}).
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
		On("WriteToUDP", body, &net.UDPAddr{}).
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
