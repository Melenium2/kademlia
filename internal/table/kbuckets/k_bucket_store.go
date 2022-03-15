package kbuckets

import (
	"time"

	"github.com/Melenium2/kademlia"
	"github.com/Melenium2/kademlia/internal/table/node"
)

type Bucket struct {
	Entries []*node.Node
}

type KBuckets struct {
	self          *node.Node
	buckets       []*Bucket
	minDistance   int
	maxBucketSize int
}

func New(self *node.Node, storageSize, minDist, maxBucketSize int) *KBuckets {
	return &KBuckets{
		self:          self,
		maxBucketSize: maxBucketSize,
		buckets:       make([]*Bucket, storageSize),
		minDistance:   minDist,
	}
}

func (kb *KBuckets) WhoAmI() *node.Node {
	return kb.self
}

func (kb *KBuckets) BucketAtDistance(dist int) *Bucket {
	if dist < kb.minDistance {
		return kb.buckets[0]
	}

	return kb.buckets[dist-kb.minDistance-1]
}

func (kb *KBuckets) BucketByID(id kademlia.ID) *Bucket {
	distance := node.LogDistance(kb.self.ID(), id)

	return kb.BucketAtDistance(distance)
}

func (kb *KBuckets) Exist(id kademlia.ID) bool {
	bucket := kb.BucketByID(id)

	return kb.ExistInBucket(bucket, id)
}

func (kb *KBuckets) ExistInBucket(bucket *Bucket, id kademlia.ID) bool {
	for _, entry := range bucket.Entries {
		if entry.ID() == id {
			return true
		}
	}

	return false
}

func (kb *KBuckets) AddSome(nodes []*node.Node) {
	for _, n := range nodes {
		kb.Add(n)
	}
}

func (kb *KBuckets) Add(node *node.Node) {
	if node.ID() == kb.self.ID() {
		return
	}

	bucket := kb.BucketByID(node.ID())

	if kb.ExistInBucket(bucket, node.ID()) {
		return
	}

	if len(bucket.Entries) >= kb.maxBucketSize {
		return
	}

	node.AddedAt(time.Now())
	bucket.Entries = append(bucket.Entries, node)
}

// TODO think about mutex in this struct. We need to secure buckets with mutex from concurrent calls.
