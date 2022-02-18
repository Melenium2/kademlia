package conn

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
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

func Marshal(fromID []byte, packet Packet) []byte {
	raw, _ := json.Marshal(packet) // nolint:errchkjson

	var (
		lenRaw  = len(raw)
		lenID   = len(fromID)
		lenType = 1
		// 5 is count of bytes: uint16 + uint16 + uint8.
		buf = bytes.NewBuffer(make([]byte, 0, lenRaw+lenID+lenType+5)) // nolint:gomnd
	)

	buf.WriteByte(byte(lenType))
	_ = binary.Write(buf, binary.LittleEndian, uint16(lenID))
	_ = binary.Write(buf, binary.LittleEndian, uint16(lenRaw))
	buf.WriteByte(packet.IAm())
	buf.Write(fromID)
	buf.Write(raw)

	return buf.Bytes()
}

func Unmarshal(raw []byte) (Packet, []byte, error) {
	if len(raw) == 0 {
		return nil, nil, ErrEmptyMessage
	}

	var (
		buf = bytes.NewBuffer(raw)

		lenType = buf.Next(1)[0]
		lenID   = binary.LittleEndian.Uint16(buf.Next(2))
		lenRaw  = binary.LittleEndian.Uint16(buf.Next(2))

		bodytype = buf.Next(int(lenType))[0]
		fromID   = buf.Next(int(lenID))
		body     = buf.Next(int(lenRaw))
	)

	var packet Packet

	switch bodytype {
	case PingMessage:
		packet = &Ping{}
	case PongMessage:
		packet = &Pong{}
	}

	if err := json.Unmarshal(body, &packet); err != nil {
		return nil, nil, err
	}

	return packet, fromID, nil
}
