package kademlia

import (
	"bytes"
	"net"

	"github.com/Melenium2/kademlia/internal/crypto"
	"github.com/Melenium2/kademlia/pkg/logger"
)

type ID [20]byte

func NewID(id [20]byte) ID {
	return id
}

func NewIDFromSlice(id []byte) ID {
	return *(*[20]byte)(id)
}

func (id *ID) Bytes() []byte {
	return id[:]
}

func (id *ID) String() string {
	return string(id.Bytes())
}

type Node struct {
	id  ID
	ip  net.IP
	udp int
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

func (n *Node) ID() ID {
	return n.id
}

func (n *Node) IP() net.IP {
	return n.ip
}

func (n *Node) UDPPort() int {
	return n.udp
}

func (n *Node) Compare(with []byte) bool {
	return bytes.Equal(n.id.Bytes(), with)
}

func GenerateID() (ID, error) {
	sha1, err := crypto.Sha1()
	if err != nil {
		return ID{}, err
	}

	p := (*ID)(sha1)

	return *p, nil
}
