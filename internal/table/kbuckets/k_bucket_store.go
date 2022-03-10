package kbuckets

import "github.com/Melenium2/kademlia/internal/table/node"

type Bucket struct {
	Entries []*node.Node
}

type KBuckets struct {
	self        *node.Node
	buckets     []*Bucket
	minDistance int
}

func New(self *node.Node, n, minDist int) *KBuckets {
	return &KBuckets{
		self:        self,
		buckets:     make([]*Bucket, n),
		minDistance: minDist,
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
