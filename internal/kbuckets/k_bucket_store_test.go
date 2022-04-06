package kbuckets_test

import (
	"math/rand"
	"net"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/Melenium2/kademlia/internal/kbuckets"
	"github.com/Melenium2/kademlia/internal/node"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var currentTime = time.Date(2022, 1, 1, 1, 1, 1, 0, time.UTC)

func TestMain(t *testing.M) {
	monkey.Patch(time.Now, func() time.Time {
		return currentTime
	})

	code := t.Run()

	monkey.Unpatch(time.Now)

	os.Exit(code)
}

var (
	id   = node.ID([20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20})
	addr = &net.UDPAddr{
		IP:   net.IPv4(1, 1, 1, 1),
		Port: 5222,
	}
	kadeNode      = node.NewNodeWithID(id, addr)
	selfNode      = kadeNode
	sixthBucketID = node.ID([20]byte{20, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20})
)

func TestKBuckets_AddOne_Should_add_new_bucket_to_all_buckets_and_set_added_at_field_to_node(t *testing.T) {
	buckets := kbuckets.New(selfNode, 15, 150, 3)

	newKadeNode := node.NewNodeWithID(sixthBucketID, addr)

	buckets.AddLocal(newKadeNode)

	b := buckets.Buckets()
	assert.Equal(t, b[6].Entries, []*node.Node{newKadeNode})
}

func TestKBuckets_AddOne_Should_not_set_provided_node_if_we_set_node_with_id_like_in_self_node(t *testing.T) {
	buckets := kbuckets.New(selfNode, 15, 150, 3)

	buckets.AddLocal(selfNode)

	b := buckets.Buckets()
	for i := range buckets.Buckets() {
		assert.Nil(t, b[i].Entries)
	}
}

func TestKBuckets_AddOne_Should_not_set_provided_node_if_node_with_same_id_existed_in_bucket(t *testing.T) {
	buckets := kbuckets.New(selfNode, 15, 150, 3)

	newKadeNode := node.NewNodeWithID(sixthBucketID, addr)
	newNode := newKadeNode

	buckets.AddLocal(newNode)
	buckets.AddLocal(newNode)

	b := buckets.Buckets()
	assert.Len(t, b[6].Entries, 1)
	assert.Equal(t, b[6].Entries, []*node.Node{newNode})
}

func TestKBuckets_AddOne_Should_not_set_node_if_len_of_buckets_reach_limit(t *testing.T) {
	buckets := kbuckets.New(selfNode, 15, 150, 1)

	newKadeNode := node.NewNodeWithID(sixthBucketID, addr)
	newNode := newKadeNode

	buckets.AddLocal(newNode)

	oldID := newNode.ID().Bytes()
	oldID[1] = 128
	newKadeNode1 := node.NewNodeWithID(node.NewIDSlice(oldID), addr)
	newNode1 := newKadeNode1

	buckets.AddLocal(newNode1)

	b := buckets.Buckets()

	for i := range b {
		if i == 6 {
			require.Len(t, b[6].Entries, 1)
			require.Equal(t, b[6].Entries, []*node.Node{newNode})

			continue
		}

		assert.Nil(t, b[i].Entries)
	}
}

func TestKBuckets_Add_Should_set_slice_of_nodes_to_buckets(t *testing.T) {
	buckets := kbuckets.New(selfNode, 15, 150, 3)

	nodes := []*node.Node{
		node.NewNode(addr), node.NewNode(addr),
		node.NewNode(addr), node.NewNode(addr),
	}

	buckets.Add(nodes)

	l := 0
	b := buckets.Buckets()

	for i := range b {
		l += len(b[i].Entries)
	}

	assert.Equal(t, 4, l)
}

func TestKBuckets_Add_Should_concurrently_accept_buckets(t *testing.T) {
	buckets := kbuckets.New(selfNode, 15, 150, 3)

	nodes := []*node.Node{
		node.NewNode(addr), node.NewNode(addr),
		node.NewNode(addr), node.NewNode(addr),
	}

	var wg sync.WaitGroup

	wg.Add(50)

	for i := 0; i < 50; i++ {
		go func() {
			buckets.Add(nodes)
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestKBuckets_FindClosest_Should_return_closest_nodes_to_provided_node(t *testing.T) {
	buckets := kbuckets.New(selfNode, 15, 150, 4)

	expected := []*node.Node{
		node.NewNode(addr), node.NewNode(addr),
		node.NewNode(addr), node.NewNode(addr),
	}

	buckets.Add(expected)

	closest := buckets.FindClosest(selfNode)

	sort.Slice(expected, func(i, j int) bool {
		return node.DistanceCmp(selfNode.ID(), expected[i].ID(), expected[j].ID()) < 0
	})

	assert.Equal(t, expected, closest)
}

func TestKBuckets_FindClosest_Should_concurrently_requests_to_kbucket_for_closest_nodes(t *testing.T) {
	buckets := kbuckets.New(selfNode, 15, 150, 3)

	nodes := []*node.Node{
		node.NewNode(addr), node.NewNode(addr),
		node.NewNode(addr), node.NewNode(addr),
	}

	buckets.Add(nodes)

	var wg sync.WaitGroup

	wg.Add(50)

	for i := 0; i < 50; i++ {
		go func() {
			buckets.FindClosest(selfNode)
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestKBuckets_MoveFront(t *testing.T) {
	bucketID1 := make([]byte, len(sixthBucketID))
	copy(bucketID1, sixthBucketID.Bytes())
	bucketID1[1] = 20

	bucketID2 := make([]byte, len(sixthBucketID))
	copy(bucketID2, sixthBucketID.Bytes())
	bucketID2[1] = 19

	sixthBucketNode1 := node.NewNodeWithID(sixthBucketID, &net.UDPAddr{})
	sixthBucketNode2 := node.NewNodeWithID(node.NewIDSlice(bucketID1), &net.UDPAddr{})
	sixthBucketNode3 := node.NewNodeWithID(node.NewIDSlice(bucketID2), &net.UDPAddr{})

	buckets := kbuckets.New(selfNode, 15, 150, 3)

	nodes := []*node.Node{
		sixthBucketNode3, sixthBucketNode2, sixthBucketNode1,
		node.NewNode(addr), node.NewNode(addr),
		node.NewNode(addr), node.NewNode(addr),
	}

	buckets.Add(nodes)

	t.Run("Should move node to front of bucket", func(t *testing.T) {
		bucket := buckets.BucketAtIndex(6)

		assert.Len(t, bucket.Entries, 3)
		assert.NotEqual(t, sixthBucketNode1, bucket.Entries[0])

		buckets.MoveFront(6, sixthBucketNode1)

		bucket = buckets.BucketAtIndex(6)

		assert.Len(t, bucket.Entries, 3)
		assert.Equal(t, sixthBucketNode1, bucket.Entries[0])
	})

	t.Run("Should move node to front from second position", func(t *testing.T) {
		bucket := buckets.BucketAtIndex(6)

		assert.Len(t, bucket.Entries, 3)
		assert.NotEqual(t, sixthBucketNode2, bucket.Entries[0])

		buckets.MoveFront(6, sixthBucketNode2)

		bucket = buckets.BucketAtIndex(6)

		assert.Len(t, bucket.Entries, 3)
		assert.Equal(t, sixthBucketNode2, bucket.Entries[0])
	})

	t.Run("Should do nothing if node at provided bucket not found", func(t *testing.T) {
		res := buckets.MoveFront(7, sixthBucketNode1)
		assert.False(t, res)
	})

	t.Run("Should concurrently request to the function", func(t *testing.T) {
		var (
			wg          sync.WaitGroup
			concurrency = 250
		)

		wg.Add(concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				curr := rand.Int31n(3)

				var n *node.Node

				switch curr {
				case 0:
					n = sixthBucketNode1
				case 1:
					n = sixthBucketNode2
				case 2:
					n = sixthBucketNode3
				}

				res := buckets.MoveFront(6, n)
				assert.True(t, res)

				wg.Done()
			}()
		}

		wg.Wait()
	})
}
