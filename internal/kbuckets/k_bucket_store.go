package kbuckets

import (
	"sync"

	"github.com/Melenium2/kademlia/internal/node"
)

type Bucket struct {
	Entries []*node.Node
}

type KBuckets struct {
	self *node.Node

	mutex   *sync.RWMutex
	buckets []*Bucket

	minDistance   int
	maxBucketSize int
}

func New(self *node.Node, storageSize, minDist, maxBucketSize int) *KBuckets {
	kb := &KBuckets{
		self:          self,
		mutex:         &sync.RWMutex{},
		maxBucketSize: maxBucketSize,
		buckets:       make([]*Bucket, storageSize),
		minDistance:   minDist,
	}

	for i := range kb.buckets {
		kb.buckets[i] = &Bucket{}
	}

	return kb
}

// FindClosest finds and returns the most closes nodes to provided node.
func (kb *KBuckets) FindClosest(node *node.Node) []*node.Node {
	nodes := NewOrderedNodes(node, kb.maxBucketSize)

	kb.mutex.RLock()

	for _, bucket := range kb.buckets {
		for _, entry := range bucket.Entries {
			nodes.Add(entry)
		}
	}

	kb.mutex.RUnlock()

	return nodes.Nodes()
}

func (kb *KBuckets) WhoAmI() *node.Node {
	return kb.self
}

func (kb *KBuckets) BucketAtDistance(dist int) *Bucket {
	if dist <= kb.minDistance {
		return kb.buckets[0]
	}

	return kb.buckets[dist-kb.minDistance-1]
}

func (kb *KBuckets) BucketByID(id node.ID) *Bucket {
	distance := node.LogDistance(kb.self.ID(), id)

	return kb.BucketAtDistance(distance)
}

func (kb *KBuckets) Exist(id node.ID) bool {
	kb.mutex.RLock()
	defer kb.mutex.RUnlock()

	bucket := kb.BucketByID(id)

	return kb.existInBucket(bucket, id)
}

func (kb *KBuckets) existInBucket(bucket *Bucket, id node.ID) bool {
	for _, entry := range bucket.Entries {
		if entry.ID() == id {
			return true
		}
	}

	return false
}

func (kb *KBuckets) Add(nodes []*node.Node) {
	kb.mutex.Lock()

	for _, n := range nodes {
		kb.add(n)
	}

	kb.mutex.Unlock()
}

func (kb *KBuckets) add(node *node.Node) {
	if node.ID() == kb.self.ID() {
		return
	}

	bucket := kb.BucketByID(node.ID())

	if kb.existInBucket(bucket, node.ID()) {
		return
	}

	if len(bucket.Entries) >= kb.maxBucketSize {
		return
	}

	bucket.Entries = append(bucket.Entries, node)
}
