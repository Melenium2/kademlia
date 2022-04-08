package kademlia

import (
	"net"
	"time"

	"github.com/Melenium2/kademlia/internal/node"
	"github.com/Melenium2/kademlia/internal/table"
)

// Config of DHT.
type Config struct {
	// ParallelNetworkCalls is a small number representing the degree of
	// parallelism in network calls, usually table.ParallelCalls, but
	// you can set it own.
	ParallelNetworkCalls int
	// Maximum bucket list size. By default, table.BucketSize.
	//
	// Bucket - is a container for nodes. Nodes stored in bucket by Kademlia
	// metric (XOR a ^ b, where a and b is ID of nodes).
	BucketSize int
	// TableRefreshRate represent time for timer, which will run node discovery
	// algorithm.
	TableRefreshRate time.Duration
	// LiveCheckRate represent time for timer, which will run node live checking
	// algorithm.
	LiveCheckRate time.Duration
}

// DHT - is distributed hash table. This hash table implements
// Kademlia DHT, and needed for creating overlap network over internet.
type DHT struct {
	table *table.Table
}

func New(_ Config, conn *net.UDPConn, knownNodesAddr []*net.UDPAddr) *DHT {
	var (
		addr       = conn.LocalAddr()
		udpAddr, _ = net.ResolveUDPAddr(addr.Network(), addr.String())
	)

	self := node.NewNode(udpAddr)
	cfg := table.Config{
		BootNodes: node.NewNodes(knownNodesAddr),
	}

	t := table.NewTable(&cfg, self, conn)

	return &DHT{
		table: t,
	}
}
