package kbuckets

import (
	"sort"

	"github.com/Melenium2/kademlia/internal/node"
)

// OrderedNodes stores Node's by their distance from our self Node.
//
// Node list has limited count. Limit is set by constructor.
type OrderedNodes struct {
	// list of nodes.
	nodes []*node.Node
	// maximum count of stored nodes.
	nodesLimit int
	// pointer to our Node.
	self *node.Node
}

// NewOrderedNodes create new instance of Node's storage.
func NewOrderedNodes(self *node.Node, limit int) OrderedNodes {
	return OrderedNodes{
		self:       self,
		nodes:      make([]*node.Node, 0, limit),
		nodesLimit: limit,
	}
}

// Add new Node to position in Node's slice by distance between
// provided Node and self Node.
func (on *OrderedNodes) Add(newNode *node.Node) {
	index := sort.Search(len(on.nodes), func(i int) bool {
		return node.DistanceCmp(on.self.ID(), on.nodes[i].ID(), newNode.ID()) > 0
	})

	// we add new item to the end of list if limit not reached.
	if len(on.nodes) < on.nodesLimit {
		on.nodes = append(on.nodes, newNode)
	}

	// if index less than length of nodes we insert it to the
	// position 'index', otherwise, 'index' equals to last element
	// of list and this is means we already set new node to
	// right position.
	if index < len(on.nodes) {
		copy(on.nodes[index+1:], on.nodes[index:])
		on.nodes[index] = newNode
	}
}

func (on *OrderedNodes) Nodes() []*node.Node {
	return on.nodes
}
