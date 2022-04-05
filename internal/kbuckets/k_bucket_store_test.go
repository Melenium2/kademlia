package kbuckets_test

import (
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
	newKadeNode1 := node.NewNodeWithID(node.NewIDFromSlice(oldID), addr)
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

func TestKBuckets_MoveFront_Should_move_node_to_front_of_bucket(t *testing.T) {

}

func TestKBuckets_MoveFront_Should_do_nothing_if_node_at_provided_bucket_not_found(t *testing.T) {

}
