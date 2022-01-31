package kademlia

import (
	"bytes"
	"net"

	"github.com/Melenium2/kademlia/internal/crypto"
)

type ID [20]byte

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

func NewNode() (*Node, error) {
	id, err := GenerateID()
	if err != nil {
		return nil, err
	}

	return &Node{
		id: id,
	}, nil
}

func (n *Node) ID() ID {
	return n.id
}

// TODO unimplemented
func (n *Node) IP() net.IP {
	return nil
}

func (n *Node) UDPPort() int {
	return n.udp
}

func (n *Node) Compare(with []byte) bool {
	return bytes.Compare(n.id.Bytes(), with) == 0
}

func GenerateID() (ID, error) {
	sha1, err := crypto.Sha1()
	if err != nil {
		return ID{}, err
	}

	p := (*ID)(sha1)

	return *p, nil
}
