package table

import (
	"github.com/Melenium2/kademlia"
	"github.com/Melenium2/kademlia/internal/table/conn"
	"github.com/Melenium2/kademlia/internal/table/kbuckets"
	"github.com/Melenium2/kademlia/internal/table/node"
)

const (
	// Alpha is a small number representing the degree of parallelism in network calls, usually 3.
	Alpha = 3
	// BucketSize is Kademlia single bucket size.
	BucketSize = 16

	hashBits          = len(kademlia.ID{}) * 8 // Length of hash in bits. Now this is length of SHA-1, 160 bits.
	nBuckets          = hashBits / 15          // Number of buckets.
	bucketMinDistance = hashBits - nBuckets    // Log distance of the closest bucket.

)

type Config struct {
	BootNodes []*kademlia.Node
}

type Table struct {
	transport *conn.Transport
	buckets   *kbuckets.KBuckets
	self      *node.Node
}

func NewTable(cfg *Config, self *kademlia.Node, connection conn.UDPConn) *Table {
	t := &Table{
		self:    node.WrapNode(self),
		buckets: kbuckets.New(node.WrapNode(self), BucketSize, bucketMinDistance),
	}

	t.transport = conn.NewTransport(connection, t.buckets)

	return t
}

// TODO in last hours, lookup struct was changed, and it already (maybe) complete
// 		todo some discover work.

func (t *Table) Discover() {
	// l := newLookup(lookupConfig{
	// 	Bootstrap: nil,
	// }, t.transport, t.self)
}

// TODO we can start implement protocol from doRefresh function
// http://xlattice.sourceforge.net/components/protocol/kademlia/specs.html#implementation
// https://en.wikipedia.org/wiki/Kademlia
// https://codethechange.stanford.edu/guides/guide_kademlia.html
