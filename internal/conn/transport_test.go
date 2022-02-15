package conn

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/Melenium2/kademlia"
	"github.com/Melenium2/kademlia/internal/table/node"
	"github.com/stretchr/testify/assert"
)

var (
	bodytype     = PongMessage
	rawPong      = []byte(`{"req_id":"MTMxMjMxMjM=","ip":"1.1.1.1","port":5222}`)
	expectedPong = &Pong{
		ReqID: []byte("13123123"),
		IP:    net.IPv4(1, 1, 1, 1),
		Port:  5222,
	}
)

func TestConsume_Should_consume_new_packet_from_result_channel_and_unmarshal_it_with_provided_type(t *testing.T) {
	var (
		packetCh = make(chan []byte, 1)
		errCh    = make(chan error, 1)
	)

	go func() {
		time.Sleep(300 * time.Millisecond)
		packetCh <- rawPong
	}()

	packet, err := consume(bodytype, packetCh, errCh)
	assert.NoError(t, err)

	pong, ok := packet.(*Pong)
	assert.True(t, ok)
	assert.Equal(t, expectedPong, pong)
}

func TestConsume_Should_consume_packet_but_got_error_while_unmarshalling(t *testing.T) {
	var (
		packetCh = make(chan []byte, 1)
		errCh    = make(chan error, 1)
		// here we got wrong type of req_id, we can not unmarshal string to slice of byte.
		internalRawPong = []byte(`{"req_id":"123123","ip":"1.1.1.1","port":5222}`)
	)

	go func() {
		time.Sleep(300 * time.Millisecond)
		packetCh <- internalRawPong
	}()

	_, err := consume(bodytype, packetCh, errCh)
	assert.Error(t, err)
}

func TestConsume_Should_got_error_while_waiting_for_new_packet(t *testing.T) {
	var (
		packetCh = make(chan []byte, 1)
		errCh    = make(chan error, 1)
	)

	go func() {
		time.Sleep(100 * time.Millisecond)
		errCh <- io.ErrClosedPipe
	}()

	_, err := consume(bodytype, packetCh, errCh)
	assert.Error(t, err)
	assert.Equal(t, io.ErrClosedPipe, err)
}

func TestTransport_PruneCall_Should_retrieve_all_items_from_rpc_channels_and_then_send_rpc_to_cancel_channel(t *testing.T) {
	rpcCall := &rpc{
		resCh: make(chan []byte, 1),
		errCh: make(chan error, 1),
	}

	rpcCall.resCh <- []byte("123123123")
	rpcCall.errCh <- io.ErrClosedPipe

	transport := Transport{
		cancelCallCh: make(chan *rpc, 1),
	}

	transport.pruneCall(rpcCall)
	assert.Len(t, rpcCall.resCh, 0)
	assert.Len(t, rpcCall.errCh, 0)
	assert.Len(t, transport.cancelCallCh, 1)
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
		rpcCall.resCh <- rawPong
	}()

	pong, err := transport.SendPing(testNode)
	assert.NoError(t, err)
	assert.Equal(t, expectedPong, pong)
	assert.Len(t, transport.cancelCallCh, 1)
}
