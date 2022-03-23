package table

import (
	"context"

	"github.com/Melenium2/kademlia/internal/conn"
	"github.com/Melenium2/kademlia/internal/kbuckets"
	"github.com/Melenium2/kademlia/internal/node"

	"github.com/Melenium2/kademlia/pkg/logger"
)

const (
	// Alpha is a small number representing the degree of parallelism in network calls, usually 3.
	Alpha = 3
	// BucketSize is Kademlia single bucket size.
	BucketSize = 16

	HashBits          = len(node.ID{}) * 8  // Length of hash in bits. Now this is length of SHA-1, 160 bits.
	NBuckets          = HashBits / 15       // Number of buckets.
	BucketMinDistance = HashBits - NBuckets // Log distance of the closest bucket.

)

type Config struct {
	BootNodes []*node.Node
}

type Table struct {
	transport *conn.Transport
	buckets   *kbuckets.KBuckets

	self           *node.Node
	bootstrapNodes []*node.Node

	log logger.Logger
}

func NewTable(cfg *Config, self *node.Node, connection conn.UDPConn) *Table {
	t := &Table{
		self:           self,
		buckets:        kbuckets.New(self, NBuckets, BucketMinDistance, BucketSize),
		bootstrapNodes: cfg.BootNodes,
		log:            logger.GetLogger(),
	}

	t.transport = conn.NewTransport(connection, t.buckets)
	go t.transport.Loop(context.Background()) // nolint:errcheck

	return t
}

func (t *Table) Discover() {
	lookupMechanism := newLookup(t.transport, t.self, lookupConfig{
		Bootstrap: t.bootstrapNodes,
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
