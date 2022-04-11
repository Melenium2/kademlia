package kademlia

import (
	"net"

	"github.com/Melenium2/kademlia/internal/node"
	"github.com/Melenium2/kademlia/internal/table"
)

// DHT - is distributed hash table. This hash table implements
// Kademlia DHT, and needed for creating overlap network over internet.
// Also, you can save anything inside DHT storage, and stored items will
// distribute over whole network.
type DHT struct {
	table *table.Table
}

func New(conn *net.UDPConn, knownNodesAddr []*net.UDPAddr, opts ...Option) *DHT {
	options := defaultOptions()

	for _, opt := range opts {
		opt(&options)
	}

	var (
		addr       = conn.LocalAddr()
		udpAddr, _ = net.ResolveUDPAddr(addr.Network(), addr.String())
	)

	self := node.NewNode(udpAddr)

	t := table.New(node.NewNodes(knownNodesAddr), self, conn, table.Config{
		ParallelCalls:    options.ParallelNetworkCalls,
		BucketSize:       options.BucketSize,
		TableRefreshRate: options.TableRefreshRate,
		LiveCheckRate:    options.LiveCheckRate,
	})

	return &DHT{
		table: t,
	}
}
