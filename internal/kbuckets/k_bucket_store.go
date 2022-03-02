package kbuckets

import "github.com/Melenium2/kademlia/internal/table/node"

type Bucket struct {
	Entries []*node.Node
}

type KBuckets struct {
	buckets     []*Bucket
	minDistance int
}

func New(n, minDist int) *KBuckets {
	return &KBuckets{
		buckets:     make([]*Bucket, n),
		minDistance: minDist,
	}
}

func (kb *KBuckets) BucketAtDistance(dist int) *Bucket {
	if dist < kb.minDistance {
		return kb.buckets[0]
	}

	return kb.buckets[dist-kb.minDistance-1]
}
