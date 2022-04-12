package kademlia

import (
	"context"
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

	// isDiscovered := t.Discover()

	return &DHT{
		table: t,
	}
}

// Maintenance start refresh / validate cycle of DHT.
//
// Function is block main thread. For closing this function need to call Close() on context
// or set Timeout or Deadline.
func (dht *DHT) Maintenance(ctx context.Context) {
	dht.table.Maintenance(ctx)
}

// ForceRefresh init force refresh cycle of DHT. This function trying to find
// closet neighbors to this node.
//
// If something goes wrong, function log issue and just return from function.
func (dht *DHT) ForceRefresh() {
	dht.table.DiscoverNeighbors()
}

// TODO add test usecase for forceRefresh
