package table

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"testing"

	"github.com/Melenium2/kademlia/internal/conn"
	"github.com/Melenium2/kademlia/internal/kbuckets"
	"github.com/Melenium2/kademlia/internal/node"

	"github.com/Melenium2/kademlia/pkg/logger"
	"github.com/stretchr/testify/require"
)

var initialPort = 36000

func transport(ctx context.Context, c *net.UDPConn, buckets conn.KBuckets) *conn.Transport {
	newTransport := conn.NewTransport(c, buckets)

	go func() {
		if err := newTransport.Loop(ctx); err != nil {
			logger.GetLogger().Error(err.Error())
		}
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

	err := initNetwork(ctx, 100)
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
