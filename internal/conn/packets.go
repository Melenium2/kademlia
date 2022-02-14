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
	reqID := make([]byte, 8)

	_, _ = rand.Read(reqID)

	return reqID
}

func Marshal(packet Packet) []byte {
	raw, _ := json.Marshal(packet)

	return raw
}

func Unmarshal(bodytype byte, raw []byte) (Packet, error) {
	var packet Packet
	switch bodytype {
	case PingMessage:
		packet = &Ping{}
	case PongMessage:
		packet = &Pong{}
	}

	if err := json.Unmarshal(raw, &packet); err != nil {
		return nil, err
	}

	return packet, nil
}
