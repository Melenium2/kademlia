package kademlia

import "time"

const (
	// ParallelCalls is a small number representing the degree of
	// parallelism in network calls, usually 3, but
	// you can set it own.
	ParallelCalls = 3
	// BucketSize is maximum bucket list size. By default, 16.
	//
	// Bucket - is a container for nodes. Nodes stored in bucket by Kademlia
	// metric (XOR a ^ b, where a and b is ID of nodes).
	BucketSize = 16
	// TableRefreshRate represent time for timer, which will run node discovery
	// algorithm.
	TableRefreshRate = 1 * time.Minute
	// LiveCheckRate represent time for timer, which will run node live checking
	// algorithm.
	LiveCheckRate = 8 * time.Second
)

// Option is a configuration setter. DHT could be configured by provided options.
type Option func(opt *Options)

// Options of DHT.
type Options struct {
	// ParallelNetworkCalls is a number of maximum concurrency call while lookup nodes process.
	//
	// By default, ParallelCalls.
	ParallelNetworkCalls int
	// BucketSize represent count of nodes containers inside DHT.
	//
	// By default, BucketSize.
	BucketSize int
	// TableRefreshRate is a time after which table will discover for new nodes.
	//
	// By default, TableRefreshRate.
	TableRefreshRate time.Duration
	// LiveCheckRate is a time after which table will ping random node. If node is unreachable
	// then table will remove it from bucket storage.
	//
	// By default, LiveCheckRate.
	LiveCheckRate time.Duration
}

func defaultOptions() Options {
	return Options{
		ParallelNetworkCalls: ParallelCalls,
		BucketSize:           BucketSize,
		TableRefreshRate:     TableRefreshRate,
		LiveCheckRate:        LiveCheckRate,
	}
}

// WithParallelCallsCount sets count of possible concurrent calls while nodes lookup process.
func WithParallelCallsCount(n int) Option {
	return func(opt *Options) {
		opt.ParallelNetworkCalls = n
	}
}

// WithBucketSize sets maximum size of bucket storage.
func WithBucketSize(n int) Option {
	return func(opt *Options) {
		opt.BucketSize = n
	}
}

// WithTableRefreshRate sets timer for discover process.
func WithTableRefreshRate(rate time.Duration) Option {
	return func(opt *Options) {
		opt.TableRefreshRate = rate
	}
}

// WithLiveCheckRate sets timer for nodes validation.
func WithLiveCheckRate(rate time.Duration) Option {
	return func(opt *Options) {
		opt.LiveCheckRate = rate
	}
}
