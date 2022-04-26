# Kademlia

Implementation of Distributed Hash table protocol in pure Go.

## Why Kademila

Kademlia DHT helps to creating overlap network. Inside this network, you may to store 
any information. All information will distribute over whole network and will be available
for all nodes by some key (we store information in hashmap).

## Example

Library has simple API for now :). 

```go
// constructor signature 
func New(port uint16, knownNodesAddr []*net.UDPAddr, opts ...Option) *DHT
```

### Default config

The `knownNodesAddr` parameter represent addresses of nodes inside network, for bootstrapping.
You should know at least one address to connect to network, otherwise, you
will be alone, for moment, while another node not connect to you.

```go
addrs := []*net.UDPAddr{
	{IP: net.IPv4(1, 1, 1, 1), Port: 5222},
}

kademliaTable := kademlia.New(15483, addrs)
```

### Additional config

Protocol has some additional user configuration. You may to change all fields to you own values if needed.

```go
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
```

In next example, we apply additional configuration to Kademlia table:

```go
opts := []kademila.Option{
	WithParallelCallsCount(4), 
	WithBucketSize(10),
	WithTableRefreshRate(1 * time.Minute),
	WithLiveCheckRate(2 * time.Minute),
}

addrs := []*net.UDPAddr{
	{IP: net.IPv4(1, 1, 1, 1), Port: 5222},
}

kademliaTable := kademlia.New(15483, addrs, opts...)
```

### Table startup

Table starts by call of `Maintenance` function, which apply some context. This function will
block thread inside which it called. For canceling this process, you need to close context, which
was passed to the function.

```go
kademliaTable := kademlia.New(15483, addrs)
ctx, cancel := context.WithCancel(context.Background())

go func() {
	if err := kademliaTable.Maintenance(); err != nil {
	    log.Print(err.Error())	
    }   
}()

// ...
// some code here.
// ...

// we close table main function.
cacnel()
```

### Force update

If you want to update table by you own, then you can call `ForseRefresh` function, which will
instantly starts new refresh table cycle.

```go
func (dht *DHT) ForceRefresh() {
    // some useful code
}
```

## Status

Work in progress.

TODO:

- Persistent storage for already found nodes, and reloading these nodes after table rerun;
- Possibility to store items inside network;
- DNS support.
- Validation of incoming IP and DNS addresses.
- New encode / decode mechanism.