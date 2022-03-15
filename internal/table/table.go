package table

import (
	"github.com/Melenium2/kademlia"
	"github.com/Melenium2/kademlia/internal/table/conn"
	"github.com/Melenium2/kademlia/internal/table/kbuckets"
	"github.com/Melenium2/kademlia/internal/table/node"
	"github.com/Melenium2/kademlia/pkg/logger"
)

const (
	// Alpha is a small number representing the degree of parallelism in network calls, usually 3.
	Alpha = 3
	// BucketSize is Kademlia single bucket size.
	BucketSize = 16

	HashBits          = len(kademlia.ID{}) * 8 // Length of hash in bits. Now this is length of SHA-1, 160 bits.
	NBuckets          = HashBits / 15          // Number of buckets.
	BucketMinDistance = HashBits - NBuckets    // Log distance of the closest bucket.

)

type Config struct {
	BootNodes []*kademlia.Node
}

type Table struct {
	transport *conn.Transport
	buckets   *kbuckets.KBuckets

	self           *node.Node
	bootstrapNodes []*kademlia.Node

	log logger.Logger
}

func NewTable(cfg *Config, self *kademlia.Node, connection conn.UDPConn) *Table {
	t := &Table{
		self:           node.WrapNode(self),
		buckets:        kbuckets.New(node.WrapNode(self), NBuckets, BucketMinDistance, BucketSize),
		bootstrapNodes: cfg.BootNodes,
		log:            logger.GetLogger(),
	}

	t.transport = conn.NewTransport(connection, t.buckets)

	return t
}

func (t *Table) Discover() {
	lookupMechanism := newLookup(t.transport, t.self, lookupConfig{
		Bootstrap: node.WrapNodes(t.bootstrapNodes),
	})

	nodes, err := lookupMechanism.Discover()
	if err != nil {
		t.log.Error(err.Error())

		return
	}

	t.buckets.Add(nodes)
}

// Docs
// http://xlattice.sourceforge.net/components/protocol/kademlia/specs.html#implementation
// https://en.wikipedia.org/wiki/Kademlia
// https://codethechange.stanford.edu/guides/guide_kademlia.html
