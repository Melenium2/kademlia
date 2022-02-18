package conn_test

import (
	"net"
	"testing"

	"github.com/Melenium2/kademlia/internal/conn"
	"github.com/stretchr/testify/assert"
)

var (
	fromID            = []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
	ping              = &conn.Ping{ReqID: []byte("13123123")}
	defaultPingBody   = []byte{0x01, 0x08, 0x00, 0x19, 0x00, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
	expectedPingBytes = append(defaultPingBody, []byte(`{"req_id":"MTMxMjMxMjM="}`)...)

	pong = &conn.Pong{
		ReqID: []byte("13123123"),
		IP:    net.IPv4(1, 1, 1, 1),
		Port:  5222,
	}
	defaultPongBody   = []byte{0x01, 0x08, 0x00, 0x34, 0x00, 0x02, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
	expectedPongBytes = append(defaultPongBody, []byte(`{"req_id":"MTMxMjMxMjM=","ip":"1.1.1.1","port":5222}`)...)
)

func TestMarshal_Should_marshal_ping_to_valid_bytes(t *testing.T) {
	res := conn.Marshal(fromID, ping)
	assert.Equal(t, expectedPingBytes, res)
}

func TestUnmarshal_Should_unmarshal_bytes_to_valid_ping_struct(t *testing.T) {
	newPing, id, err := conn.Unmarshal(expectedPingBytes)
	assert.NoError(t, err)
	assert.Equal(t, ping, newPing)
	assert.Equal(t, fromID, id)
}

func TestMarshal_Should_marshal_pong_to_valid_bytes(t *testing.T) {
	res := conn.Marshal(fromID, pong)
	assert.Equal(t, expectedPongBytes, res)
}

func TestUnmarshal_Should_unmarshal_bytes_to_valid_pong_struct(t *testing.T) {
	newPong, id, err := conn.Unmarshal(expectedPongBytes)
	assert.NoError(t, err)
	assert.Equal(t, pong, newPong)
	assert.Equal(t, fromID, id)
}
