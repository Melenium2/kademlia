Initially we need to create table

Then we crate table we set self-node as parameter to constructor
of table.

Then we get from storage random nodes and used it as bootstrap
nodes. All bootstrap nodes we set to special container which hold 
all bootstrap nodes.

We need to do lookup of neighbors. We already get random neighbors, 
and ready to start from them.

1) Create lookup struct, which store already checked neighbors and
    neighbors that we need to check.
2) We process "lookup" until all nodes asked.


Lookup algo:
* Set bootstrap nodes.
* Set that our self node already existed and processed
* Query to all bootstrap nodes in loop
* Add entries to some storage and each entry should be sorted 
    by distance.
* Return found nodes.