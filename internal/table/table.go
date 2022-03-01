package table

import (
	"github.com/Melenium2/kademlia"
	"github.com/Melenium2/kademlia/internal/table/node"
)

const (
	// Alpha is a small number representing the degree of parallelism in network calls, usually 3.
	Alpha = 3
	// BucketSize is Kademlia single bucket size.
	BucketSize = 16

	hashBits = len(kademlia.ID{}) * 8 // Length of hash in bits. Now this is length of SHA-1, 160 bits.
	nBuckets = hashBits / 15          // Number of buckets.
	// nolint:unused,deadcode,varcheck
	bucketMinDistance = hashBits - nBuckets // Log distance of the closest bucket.

)

type BucketStorage interface {
}

type Config struct {
	BootNodes []*kademlia.Node
}

// nolint:unused,structcheck
type bucket struct {
	entries      []*node.Node // live entries, sorted by time of last ping.
	replacements []*node.Node // if one of entries is dies, then we chose one from this list, if it is still available.
}

// nolint:structcheck,unused
type Table struct {
	store   BucketStorage
	buckets []*bucket
	self    *node.Node
}

func NewTable(cfg *Config, db BucketStorage, kadenode *kademlia.Node) *Table {
	return &Table{
		store: db,
		self:  node.WrapNode(kadenode),
	}
}

func (t *Table) Discover() {}

// TODO we can start implement protocol from doRefresh function
// http://xlattice.sourceforge.net/components/protocol/kademlia/specs.html#implementation
// https://en.wikipedia.org/wiki/Kademlia
// https://codethechange.stanford.edu/guides/guide_kademlia.html
