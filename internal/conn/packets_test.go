package conn_test

import (
	"net"
	"testing"

	"github.com/Melenium2/kademlia/internal/conn"
	"github.com/stretchr/testify/assert"
)

var (
	ping              = &conn.Ping{ReqID: []byte("13123123")}
	expectedPingBytes = append([]byte{0x1}, []byte(`{"req_id":"MTMxMjMxMjM="}`)...)

	pong = &conn.Pong{
		ReqID: []byte("13123123"),
		IP:    net.IPv4(1, 1, 1, 1),
		Port:  5222,
	}
	expectedPongBytes = append([]byte{0x02}, []byte(`{"req_id":"MTMxMjMxMjM=","ip":"1.1.1.1","port":5222}`)...)
)

func TestMarshal_Should_marshal_ping_to_valid_bytes(t *testing.T) {
	res := conn.Marshal(ping)
	assert.Equal(t, expectedPingBytes, res)
}

func TestUnmarshal_Should_unmarshal_bytes_to_valid_ping_struct(t *testing.T) {
	newPing, err := conn.Unmarshal(expectedPingBytes)
	assert.NoError(t, err)
	assert.Equal(t, ping, newPing)
}

func TestMarshal_Should_marshal_pong_to_valid_bytes(t *testing.T) {
	res := conn.Marshal(pong)
	assert.Equal(t, expectedPongBytes, res)
}

func TestUnmarshal_Should_unmarshal_bytes_to_valid_pong_struct(t *testing.T) {
	newPong, err := conn.Unmarshal(expectedPongBytes)
	assert.NoError(t, err)
	assert.Equal(t, pong, newPong)
}
