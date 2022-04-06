package node

import (
	"net"
	"time"

	"github.com/Melenium2/kademlia/pkg/logger"
)

type ID [20]byte

func NewID(id [20]byte) ID {
	return id
}

func NewIDSlice(id []byte) ID {
	return *(*[20]byte)(id)
}

func (id ID) Bytes() []byte {
	return id[:]
}

func (id ID) String() string {
	return string(id.Bytes())
}

type Node struct {
	id      ID
	ip      net.IP
	udp     int
	addedAt time.Time
}

func NewNode(addr *net.UDPAddr) *Node {
	id, err := GenerateID()
	if err != nil {
		logger.GetLogger().Fatal(err.Error())
	}

	return &Node{
		id:  id,
		ip:  addr.IP,
		udp: addr.Port,
	}
}

func NewNodeWithID(id ID, addr *net.UDPAddr) *Node {
	return &Node{
		id:  id,
		ip:  addr.IP,
		udp: addr.Port,
	}
}

func NewNodeFromScratch(id ID, ip net.IP, port int, addedAt time.Time) *Node {
	return &Node{
		id:      id,
		ip:      ip,
		udp:     port,
		addedAt: addedAt,
	}
}

func (n *Node) Addr() net.UDPAddr {
	return net.UDPAddr{
		IP:   n.IP(),
		Port: n.UDPPort(),
	}
}

func (n *Node) AddedAt(time time.Time) {
	n.addedAt = time
}

func (n *Node) AddedTime() time.Time {
	return n.addedAt
}

func (n Node) ID() ID {
	return n.id
}

func (n Node) IP() net.IP {
	return n.ip
}

func (n Node) UDPPort() int {
	return n.udp
}

func (n *Node) IsEqual(node *Node) bool {
	if n.id != node.id {
		return false
	}

	if !n.ip.Equal(node.ip) {
		return false
	}

	if n.udp != node.udp {
		return false
	}

	return true
}
