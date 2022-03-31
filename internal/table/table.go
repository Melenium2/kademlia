package table

import (
	"context"
	"crypto/rand"
	"net"

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
	buckets := kbuckets.New(self, NBuckets, BucketMinDistance, BucketSize)
	transport := conn.NewTransport(connection, buckets)

	go transport.Loop(context.Background()) // nolint:errcheck

	t := &Table{
		transport:      transport,
		self:           self,
		buckets:        buckets,
		bootstrapNodes: cfg.BootNodes,
		log:            logger.GetLogger(),
	}

	return t
}

func (t *Table) Discover() {
	nodes, err := t.discover(t.bootstrapNodes)
	if err != nil {
		t.log.Error(err.Error())

		return
	}

	t.buckets.Add(nodes)
}

func (t *Table) discover(nodes []*node.Node) ([]*node.Node, error) {
	lookupMechanism := newLookup(t.transport, t.self, lookupConfig{
		Bootstrap: nodes,
	})

	return lookupMechanism.Discover()
}

func (t *Table) findNeighbors(node *node.Node) ([]*node.Node, error) {
	closest := t.buckets.FindClosest(node)

	return t.discover(closest)
}

func (t *Table) findRandom() ([]*node.Node, error) {
	var randID node.ID
	_, _ = rand.Read(randID[:])

	n := node.NewNodeWithID(randID, &net.UDPAddr{})

	return t.findNeighbors(n)
}

// Docs
// http://xlattice.sourceforge.net/components/protocol/kademlia/specs.html#implementation
// https://en.wikipedia.org/wiki/Kademlia
// https://codethechange.stanford.edu/guides/guide_kademlia.html
// https://folk.universitetetioslo.no/michawe/teaching/p2p-ws08/p2p-5-6.pdf
