package table

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"testing"

	"github.com/Melenium2/kademlia/internal/conn"
	"github.com/Melenium2/kademlia/internal/conn/mocks"
	"github.com/Melenium2/kademlia/internal/kbuckets"
	"github.com/Melenium2/kademlia/internal/node"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var initialPort = 36000

func transport(ctx context.Context, c *net.UDPConn, buckets conn.KBuckets) *conn.Transport {
	newTransport := conn.NewTransport(c, buckets)

	go func() {
		_ = newTransport.Loop(ctx)
	}()

	return newTransport
}

func randomPort(from, to int) int {
	return int(rand.Int63n(int64(to-from))) + from
}

func resolveAddr(port int) (*net.UDPAddr, error) {
	return net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", port))
}

func udpConn(addr *net.UDPAddr) (*net.UDPConn, error) {
	c, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func bootstrapNodes(ports ...int) []*node.Node {
	nodes := make([]*node.Node, len(ports))

	for i := 0; i < len(ports); i++ {
		addr, _ := resolveAddr(ports[i])

		nodes[i] = node.NewNode(addr)
	}

	return nodes
}

func initNetwork(ctx context.Context, nodesCount int) error {
	usedPorts := make(map[int]struct{}, nodesCount)
	nodes := make([]*node.Node, 0, nodesCount)
	stores := make(map[int]*kbuckets.KBuckets, nodesCount)

	for i := 0; i < nodesCount; i++ {
		port := randomPort(initialPort, initialPort+nodesCount)
		if _, ok := usedPorts[port]; ok {
			continue
		}

		addr, _ := resolveAddr(port)
		c, err := udpConn(addr)
		if err != nil {
			return err
		}

		selfNode := node.NewNode(addr)

		nodes = append(nodes, selfNode)

		store := kbuckets.New(selfNode, NBuckets, BucketMinDistance, BucketSize)
		stores[port] = store

		transport(ctx, c, store)

		usedPorts[port] = struct{}{}
	}

	for _, buckets := range stores {
		buckets.Add(nodes)
	}

	return nil
}

func TestTable_Discover_Should_find_nodes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := initNetwork(ctx, 300)
	require.NoError(t, err)

	cfg := Config{
		BootNodes: bootstrapNodes(initialPort+1, initialPort+2, initialPort+3),
	}

	addr, _ := resolveAddr(15000)
	self := node.NewNode(addr)
	c, err := udpConn(addr)
	require.NoError(t, err)

	table := NewTable(&cfg, self, c)

	table.Discover()
}

var fakeConn = func() *mocks.UDPConn {
	fake := &mocks.UDPConn{}
	fake.On("ReadFromUDP", mock.IsType([]byte{})).Return(0, &net.UDPAddr{}, nil)

	return fake
}

func TestTable_NodeValidation_Should_ping_to_found_node_and_move_it_to_front(t *testing.T) {

}

func TestTable_NodeValidation_Should_do_nothing_if_can_not_found_last_wrote_node_in_buckets(t *testing.T) {

}

func TestTable_NodeValidation_Should_remove_node_from_bucket_if_can_not_ping_it(t *testing.T) {

}

func TestTable_DeleteLastNode_Should_remove_node_with_provided_id_from_bucket_at_specific_index(t *testing.T) {
	var (
		selfNode  = node.NewNode(&net.UDPAddr{})
		table     = NewTable(&Config{BootNodes: []*node.Node{}}, selfNode, fakeConn())
		firstNode = node.NewNode(&net.UDPAddr{Port: 5222})
	)

	table.buckets.Add([]*node.Node{
		node.NewNode(&net.UDPAddr{}), node.NewNode(&net.UDPAddr{}), node.NewNode(&net.UDPAddr{}),
		node.NewNode(&net.UDPAddr{}), node.NewNode(&net.UDPAddr{}), node.NewNode(&net.UDPAddr{}),
		firstNode,
	})

	// find bucket which contains firstNode.
	bucketIndex := node.LogDistance(selfNode.ID(), firstNode.ID()) - BucketMinDistance - 1

	deleted := table.deleteLastNode(bucketIndex, firstNode)
	assert.True(t, deleted)
}

func TestTable_DeleteLastNode_Should_do_nothing_if_bucket_at_provided_index_equals_to_nil(t *testing.T) {
	var (
		selfNode  = node.NewNode(&net.UDPAddr{})
		table     = NewTable(&Config{BootNodes: []*node.Node{}}, selfNode, fakeConn())
		firstNode = node.NewNode(&net.UDPAddr{Port: 5222})
	)

	table.buckets.Add([]*node.Node{
		node.NewNode(&net.UDPAddr{}), node.NewNode(&net.UDPAddr{}), node.NewNode(&net.UDPAddr{}),
		node.NewNode(&net.UDPAddr{}), node.NewNode(&net.UDPAddr{}), node.NewNode(&net.UDPAddr{}),
		firstNode,
	})

	deleted := table.deleteLastNode(0, firstNode)
	assert.False(t, deleted)
}
