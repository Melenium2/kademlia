package kademlia_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/Melenium2/kademlia"
)

// TODO may be init addresses as nodes for tests.

var (
	known1, _ = net.ResolveUDPAddr("udp", ":2222")
	known2, _ = net.ResolveUDPAddr("udp", ":3333")
	known3, _ = net.ResolveUDPAddr("udp", ":4444")
	knwons    = []*net.UDPAddr{known1, known2, known3}
)

func TestDHT_Maintenance_Should_create_and_dht_table(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	dht := kademlia.New(1111, knwons)

	_ = dht.Maintenance(ctx)
}

func TestDHT_Maintenance_Should_create_new_dht_with_options(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	dht := kademlia.New(
		1112,
		knwons,
		kademlia.WithLiveCheckRate(10*time.Second),
		kademlia.WithTableRefreshRate(1*time.Minute),
		kademlia.WithBucketSize(16),
		kademlia.WithParallelCallsCount(3),
	)

	_ = dht.Maintenance(ctx)
}
