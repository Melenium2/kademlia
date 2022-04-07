# Kademlia

This is implementation of Distributed Hash table protocol in pure Go.

## Status

Already completed:

- Ping - Pong requests, now table can check is nodes inside buckets still alive;
- FindNode request, table can doing self-updating.

TODO:

- Persistent storage for already found nodes, and reloading these nodes after table rerun;
- Store some items inside network;
- DNS support.