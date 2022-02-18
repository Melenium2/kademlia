package conn

import (
	"crypto/rand"
	"encoding/json"
	"net"
)

const (
	PingMessage byte = iota + 1
	PongMessage
)

type Packet interface {
	SetRequestID(id []byte)
	Name() string
	IAm() byte
}

type Ping struct {
	ReqID []byte `json:"req_id,omitempty"`
}

func (p *Ping) SetRequestID(id []byte) {
	p.ReqID = id
}

func (p *Ping) Name() string {
	return "PING"
}

func (p *Ping) IAm() byte {
	return PingMessage
}

type Pong struct {
	ReqID []byte `json:"req_id,omitempty"`
	IP    net.IP `json:"ip,omitempty"`
	Port  uint16 `json:"port,omitempty"`
}

func (p *Pong) SetRequestID(id []byte) {
	p.ReqID = id
}

func (p *Pong) Name() string {
	return "PONG"
}

func (p *Pong) IAm() byte {
	return PongMessage
}

func GenerateReqID() []byte {
	// TODO mb move ID len to cons
	reqID := make([]byte, 8) // nolint:gomnd

	_, _ = rand.Read(reqID)

	return reqID
}

func Marshal(packet Packet) []byte {
	raw, _ := json.Marshal(packet) // nolint:errchkjson

	marshaled := make([]byte, len(raw)+1)
	copy(marshaled[1:], raw)

	marshaled[0] = packet.IAm()

	return marshaled
}

func Unmarshal(raw []byte) (Packet, error) {
	if len(raw) == 0 {
		return nil, ErrEmptyMessage
	}

	bodytype := raw[0]

	var packet Packet

	switch bodytype {
	case PingMessage:
		packet = &Ping{}
	case PongMessage:
		packet = &Pong{}
	}

	if err := json.Unmarshal(raw[1:], &packet); err != nil {
		return nil, err
	}

	return packet, nil
}
