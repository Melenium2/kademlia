package kbuckets

import (
	"github.com/Melenium2/kademlia"
	"github.com/Melenium2/kademlia/internal/table/node"
)

func (kb *KBuckets) ExistInBucket(bucket *Bucket, id kademlia.ID) bool {
	return kb.existInBucket(bucket, id)
}

func (kb *KBuckets) AddLocal(node *node.Node) {
	kb.add(node)
}

func (kb *KBuckets) Buckets() []*Bucket {
	return kb.buckets
}
