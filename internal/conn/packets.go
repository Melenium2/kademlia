package conn

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"net"

	"github.com/Melenium2/kademlia/internal/table/node"
)

const (
	PingMessage byte = iota + 1
	PongMessage
	FindNodesMessage
	NodesListMessage
)

type Packet interface {
	GetRequestID() []byte
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

func (p *Ping) GetRequestID() []byte {
	return p.ReqID
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

func (p *Pong) GetRequestID() []byte {
	return p.ReqID
}

func (p *Pong) Name() string {
	return "PONG"
}

func (p *Pong) IAm() byte {
	return PongMessage
}

type FindNodes struct {
	ReqID     []byte
	Distances []uint
}

func (fn *FindNodes) SetRequestID(id []byte) {
	fn.ReqID = id
}

func (fn *FindNodes) GetRequestID() []byte {
	return fn.ReqID
}

func (fn *FindNodes) Name() string {
	return "FIND_NODE"
}

func (fn *FindNodes) IAm() byte {
	return FindNodesMessage
}

type NodesList struct {
	ReqID []byte
	Count uint8
	Nodes []*node.Node
}

func (fn *NodesList) SetRequestID(id []byte) {
	fn.ReqID = id
}

func (fn *NodesList) GetRequestID() []byte {
	return fn.ReqID
}

func (fn *NodesList) Name() string {
	return "NODES_LIST"
}

func (fn *NodesList) IAm() byte {
	return NodesListMessage
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
	case FindNodesMessage:
		packet = &FindNodes{}
	case NodesListMessage:
		packet = &NodesList{}
	}

	if err := json.Unmarshal(body, &packet); err != nil {
		return nil, nil, err
	}

	return packet, fromID, nil
}
