package table

import (
	"net"
	"sort"

	"github.com/Melenium2/kademlia"
	"github.com/Melenium2/kademlia/internal/table/node"
	"github.com/Melenium2/kademlia/pkg/logger"
)

// orderedNodes stores Node's by their distance from our self Node.
//
// Node list has limited count. Limit is set by constructor.
type orderedNodes struct {
	// list of nodes.
	nodes []*node.Node
	// maximum count of stored nodes.
	nodesLimit int
	// pointer to our Node.
	self *node.Node
}

// newOrderedNodes create new instance of Node's storage.
func newOrderedNodes(self *node.Node, limit int) orderedNodes {
	return orderedNodes{
		self:       self,
		nodes:      make([]*node.Node, 0, limit),
		nodesLimit: limit,
	}
}

// Add new Node to position in Node's slice by distance between
// provided Node and self Node.
func (on *orderedNodes) Add(newNode *node.Node) {
	index := sort.Search(len(on.nodes), func(i int) bool {
		return node.DistanceCmp(on.self.ID(), on.nodes[i].ID(), newNode.ID()) > 0
	})

	// we add new item to the end of list if limit not reached.
	if len(on.nodes) < on.nodesLimit {
		on.nodes = append(on.nodes, newNode)
	}

	// if index less than length of nodes we insert it to the
	// position 'index', otherwise, 'index' equals to last element
	// of list and this is means we already set new node to
	// right position.
	if index < len(on.nodes) {
		copy(on.nodes[index+1:], on.nodes[index:])
		on.nodes[index] = newNode
	}
}

func (on *orderedNodes) Nodes() []*node.Node {
	return on.nodes
}

// Finder is interface which provide functionality to network find possibility.
// Finder try to connect to Node with settings provided by the arguments
// and return the closest nodes to provided Node.
type finder interface {
	Find(id kademlia.ID, ip net.IP) ([]*node.Node, error)
}

// lookupConfig is configuration to lookup mechanism.
type lookupConfig struct {
	Bootstrap []*node.Node
}

// lookup mechanism that trying to found all Node's in the network that
// close to our self Node. It approaches the target by querying nodes that
// are closer to it on each iteration.
type lookup struct {
	// finder dials net calls to other nodes by their IP and ID.
	finder finder
	log    logger.Logger

	// nodes which already asked for new closer nodes.
	askedNodes map[kademlia.ID]struct{}
	// nodes which was already processed and stored in resultNodes.
	seenNodes map[kademlia.ID]struct{}
	// result of lookup request. This structure will sort nodes
	// by their distance between self node and other nodes.
	resultNodes orderedNodes
	// nodes for initializing lookup process. At least one node needed for
	// starting process.
	bootstrap []*node.Node
	// count of concurrent searches. If started equals to zero this is
	// means that we already looked at each node in network.
	started int
}

// newLookup create new lookup mechanism.
func newLookup(cfg lookupConfig, finder finder, self *node.Node) *lookup {
	return &lookup{
		finder:      finder,
		log:         logger.GetLogger(),
		askedNodes:  make(map[kademlia.ID]struct{}),
		seenNodes:   make(map[kademlia.ID]struct{}),
		resultNodes: newOrderedNodes(self, BucketSize),
		bootstrap:   cfg.Bootstrap,
	}
}

// Discover starting search mechanism. This function will block main thread
// until searching not completed. Discover will find all Node's in network that
// close to our self Node and return it in sorted order. Items will be sorted
// by the distance between our Node and searched Node's.
func (l *lookup) Discover() ([]*node.Node, error) {
	if len(l.bootstrap) == 0 {
		return nil, ErrEmptyBootstrapNodes
	}

	for i := 0; i < len(l.bootstrap); i++ {
		l.resultNodes.Add(l.bootstrap[i])
	}

	var (
		resCh = make(chan []*node.Node, Alpha)
		errCh = make(chan error, 1)
	)

	if err := l.start(resCh, errCh); err != nil {
		return nil, err
	}

	close(resCh)
	close(errCh)

	l.drainCh(resCh, errCh)

	return l.resultNodes.Nodes(), nil
}

// drainCh reads all remaining items from channels. Channels should be closed,
// otherwise, this function will block current thread.
func (l *lookup) drainCh(resCh chan []*node.Node, errCh chan error) {
	for res := range resCh {
		l.appendNodes(res)
	}

	for err := range errCh {
		l.log.Errorf("error occurs while scan nodes %s", err)
	}
}

// appendNodes to the resulting structure if no nodes are viewed.
func (l *lookup) appendNodes(nodes []*node.Node) {
	for _, n := range nodes {
		if n == nil {
			continue
		}

		if _, ok := l.seenNodes[n.ID()]; ok {
			continue
		}

		l.resultNodes.Add(n)
		l.seenNodes[n.ID()] = struct{}{}
	}
}

// consume wait for first reply from one of provided channels.
func (l *lookup) consume(resCh chan []*node.Node, errCh chan error) error {
	select {
	case nodes := <-resCh:
		l.appendNodes(nodes)
		l.started--

		return nil
	case err := <-errCh:
		l.started--

		return err
	}
}

// start 'discover' algorithm. In every iteration we take only closest nodes we have not asked yet.
// if lookup.started parameter equals to 0, this is means that we ended discovery nad lookup.resultNodes
// contains the closest nodes to our self Node.
func (l *lookup) start(resCh chan []*node.Node, errCh chan error) error {
	for l.started >= 0 {
		// we loop over all nodes and query it for closest nodes. We can not
		// run more parallel scans than Alpha (3).
		for i := 0; i < len(l.resultNodes.Nodes()) && l.started < Alpha; i++ {
			curr := l.resultNodes.nodes[i]

			if _, ok := l.askedNodes[curr.ID()]; ok {
				continue
			}

			l.askedNodes[curr.ID()] = struct{}{}
			l.started++

			go l.scan(curr, resCh, errCh)
		}

		// if we do not start any find cycle, this is means that we already find
		// all nodes and should cancel 'discover' cycle. One more reason why we
		// stop here if started flag equals zero, this is deadlock, which occurs
		// if we call 'consume' function here, because no result incoming.
		if l.started == 0 {
			// decrement value to close cycle.
			l.started--

			continue
		}

		if err := l.consume(resCh, errCh); err != nil {
			return err
		}
	}

	return nil
}

// scan provided Node to find the closest nodes to our self Node.
func (l *lookup) scan(curr *node.Node, resCh chan []*node.Node, errCh chan error) {
	nodes, err := l.finder.Find(curr.ID(), curr.IP())
	if err != nil {
		errCh <- err

		return
	}

	resCh <- nodes
}
