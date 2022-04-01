package table

import (
	"context"
	crand "crypto/rand"
	"math/rand"
	"net"
	"time"

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

	RefreshRate   = 1 * time.Minute
	LiveCheckRate = 25 * time.Second
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
	_, _ = crand.Read(randID[:])

	n := node.NewNodeWithID(randID, &net.UDPAddr{})

	return t.findNeighbors(n)
}

func (t *Table) Maintenance(ctx context.Context) {
	var (
		refreshTicker = time.NewTicker(RefreshRate)
		pingTicker    = time.NewTicker(LiveCheckRate)
	)

	for {
		select {
		case <-refreshTicker.C:
		case <-pingTicker.C:
			go t.NodeValidation()
		case <-ctx.Done():
			refreshTicker.Stop()
			pingTicker.Stop()

			return
		}
	}
}

func (t *Table) NodeValidation() {
	perm := rand.Perm(NBuckets)

	var (
		bucketIndex int
		bucketNode  *node.Node
	)

	// find first not empty bucket.
	for _, index := range perm {
		bucket := t.buckets.BucketAtIndex(index)

		if len(bucket.Entries) > 0 {
			bucketNode = bucket.Entries[len(bucket.Entries)-1]
			bucketIndex = index

			break
		}
	}

	if bucketNode == nil {
		return
	}

	_, err := t.transport.SendPing(bucketNode)
	if err != nil {
		t.deleteLastNode(bucketIndex, bucketNode)

		t.log.Errorf(
			"can not send ping to node (%s) with addr %s, remove it from bucket %d",
			bucketNode.ID(), bucketNode.Addr(), bucketIndex,
		)

		return
	}

	t.buckets.MoveFront(bucketIndex, bucketNode)
}

func (t *Table) deleteLastNode(bucketIndex int, node *node.Node) {
	bucket := t.buckets.BucketAtIndex(bucketIndex)
	last := bucket.Entries[len(bucket.Entries)-1]

	// if bucket already empty, or last node not equals to provided node, return from func.
	if len(bucket.Entries) == 0 || last.ID() != node.ID() {
		return
	}

	t.buckets.Delete(bucketIndex, node)
}

func (t *Table) DiscoverNeighbors() {}

// Docs
// http://xlattice.sourceforge.net/components/protocol/kademlia/specs.html#implementation
// https://en.wikipedia.org/wiki/Kademlia
// https://codethechange.stanford.edu/guides/guide_kademlia.html
// https://folk.universitetetioslo.no/michawe/teaching/p2p-ws08/p2p-5-6.pdf
