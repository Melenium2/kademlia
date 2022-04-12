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

// TODO
//		Add client code for running table
//		Add config variables
//		Change encode/decode method
//      Now we can not start out table without any bootstrap nodes,
//		we should started without nodes too.

const (
	HashBits          = len(node.ID{}) * 8  // Length of hash in bits. Now this is length of SHA-1, 160 bits.
	NBuckets          = HashBits / 15       // Number of buckets.
	BucketMinDistance = HashBits - NBuckets // Log distance of the closest bucket.
)

type Config struct {
	ParallelCalls    int
	BucketSize       int
	TableRefreshRate time.Duration
	LiveCheckRate    time.Duration
}

type Table struct {
	transport *conn.Transport
	buckets   *kbuckets.KBuckets

	self           *node.Node
	bootstrapNodes []*node.Node

	cfg Config
	log logger.Logger
}

func New(bootNodes []*node.Node, self *node.Node, connection conn.UDPConn, cfg Config) *Table {
	buckets := kbuckets.New(self, NBuckets, BucketMinDistance, cfg.BucketSize)
	transport := conn.NewTransport(connection, buckets)

	go transport.Loop(context.Background()) // nolint:errcheck

	t := &Table{
		transport:      transport,
		self:           self,
		buckets:        buckets,
		bootstrapNodes: bootNodes,
		cfg:            cfg,
		log:            logger.GetLogger(),
	}

	return t
}

// Discover will request to bootstrap nodes, provided by config, for
// nodes in the network.
//
// If bootstrap nodes, in the configuration, are empty, this function log error and return false.
func (t *Table) Discover() bool {
	nodes, err := t.discover(t.bootstrapNodes)
	if err != nil {
		t.log.Error(err.Error())

		return false
	}

	t.buckets.Add(nodes)

	return true
}

func (t *Table) discover(nodes []*node.Node) ([]*node.Node, error) {
	lookupMechanism := newLookup(t.transport, t.self, lookupConfig{
		Bootstrap:       nodes,
		ConcurrentCalls: t.cfg.ParallelCalls,
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

// Maintenance start table update cycle. While cycle, table will find new nodes and
// checks already added nodes for availability. For closing cycle you must cancel context.Context.
func (t *Table) Maintenance(ctx context.Context) {
	var (
		refreshTicker = time.NewTicker(t.cfg.TableRefreshRate)
		pingTicker    = time.NewTicker(t.cfg.LiveCheckRate)
	)

	for {
		select {
		case <-refreshTicker.C:
			go t.DiscoverNeighbors()
		case <-pingTicker.C:
			go t.NodeValidation()
		case <-ctx.Done():
			refreshTicker.Stop()
			pingTicker.Stop()

			return
		}
	}
}

// NodeValidation checks nodes inside buckets that they still 'alive', otherwise remove it.
//
// How it works:
// NodeValidation gets last node from random bucket and checks that node still alive.
// If node not reply to the ping message, then it's removes from bucket, otherwise node
// consider as checked and validated, and moves to front of these random bucket.
func (t *Table) NodeValidation() {
	timer := t.timer("validation", time.Now())
	defer timer()

	perm := rand.Perm(NBuckets)

	var (
		bucketIndex int
		bucketNode  *node.Node
	)

	// find first not empty bucket.
	for _, index := range perm {
		bucket := t.buckets.BucketAtIndex(index)

		if len(bucket.Entries) > 0 {
			// take oldest wrote entry.
			bucketNode = bucket.Entries[len(bucket.Entries)-1]
			bucketIndex = index

			break
		}
	}

	if bucketNode == nil {
		t.log.Warn("no nodes found in the buckets")

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

	t.log.Infof("node at address %s:%d still alive", bucketNode.Addr().IP, bucketNode.Addr().Port)

	t.buckets.MoveFront(bucketIndex, bucketNode)
}

func (t *Table) deleteLastNode(bucketIndex int, node *node.Node) bool {
	bucket := t.buckets.BucketAtIndex(bucketIndex)

	if bucket == nil || len(bucket.Entries) == 0 {
		return false
	}

	last := bucket.Entries[len(bucket.Entries)-1]

	// if bucket already empty, or last node not equals to provided node, return from func.
	if last.ID() != node.ID() {
		return false
	}

	t.buckets.Delete(bucketIndex, node)

	return true
}

// DiscoverNeighbors finds new neighbors nodes close to self-node, and make fast random lookups
// by some random nodes.
func (t *Table) DiscoverNeighbors() {
	timer := t.timer("discover", time.Now())
	defer timer()

	neighbors, err := t.findNeighbors(t.self)
	if err != nil {
		t.log.Error(err.Error())

		return
	}

	t.buckets.Add(neighbors)

	for i := 0; i < 5; i++ {
		random, err := t.findRandom()
		if err != nil {
			t.log.Error(err.Error())

			return
		}

		t.buckets.Add(random)
	}
}

func (t *Table) timer(operation string, now time.Time) func() {
	return func() {
		t.log.Infof("%s time progress = %s", operation, time.Since(now))
	}
}

// Docs
// http://xlattice.sourceforge.net/components/protocol/kademlia/specs.html#implementation
// https://en.wikipedia.org/wiki/Kademlia
// https://codethechange.stanford.edu/guides/guide_kademlia.html
// https://folk.universitetetioslo.no/michawe/teaching/p2p-ws08/p2p-5-6.pdf
