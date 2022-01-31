package node

import (
	"net"

	"github.com/Melenium2/kademlia"
)

type Node struct {
	*kademlia.Node
}

func (n *Node) Addr() net.UDPAddr {
	return net.UDPAddr{
		IP:   n.IP(),
		Port: n.UDPPort(),
	}
}

func WrapNode(n *kademlia.Node) *Node {
	return &Node{
		n,
	}
}

// DistanceCmp compares the distance between self-Node and Node-a, also,
// self-Node and Node-b. Function returns -1 if a closer to self-Node, 1 if b
// closer to self Node and 0 if distances are equal.
func DistanceCmp(self, a, b kademlia.ID) int {
	for i := 0; i < len(self); i++ {
		distA := a[i] ^ self[i]
		distB := b[i] ^ self[i]

		if distA > distB {
			return 1
		} else if distA < distB {
			return -1
		}
	}

	return 0
}
