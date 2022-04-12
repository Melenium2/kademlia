package kademlia

import (
	"context"
	"fmt"
	"net"

	"github.com/Melenium2/kademlia/internal/node"
	"github.com/Melenium2/kademlia/internal/table"
	"github.com/Melenium2/kademlia/pkg/logger"
)

// DHT - is distributed hash table. This hash table implements
// Kademlia DHT, and needed for creating overlap network over internet.
// Also, you can save anything inside DHT storage, and stored items will
// distribute over whole network.
type DHT struct {
	table *table.Table
}

func New(port uint16, knownNodesAddr []*net.UDPAddr, opts ...Option) *DHT {
	options := defaultOptions()

	for _, opt := range opts {
		opt(&options)
	}

	var (
		portStr    = fmt.Sprintf(":%d", port)
		udpAddr, _ = net.ResolveUDPAddr("udp", portStr)
	)

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		logger.GetLogger().Fatal(err.Error())
	}

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

// Maintenance start refresh / validate cycle of DHT.
//
// Function is block main thread. For closing this function need to call Close() on context
// or set Timeout or Deadline.
func (dht *DHT) Maintenance(ctx context.Context) error {
	isDiscovered := dht.table.Discover()
	if !isDiscovered {
		return fmt.Errorf("%w on table init", ErrDiscover)
	}

	dht.table.Maintenance(ctx)

	return nil
}

// ForceRefresh init force refresh cycle of DHT. This function trying to find
// closet neighbors to this node.
//
// If something goes wrong, function log issue and just return from function.
func (dht *DHT) ForceRefresh() {
	dht.table.DiscoverNeighbors()
}
