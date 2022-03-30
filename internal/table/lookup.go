package table

import (
	"github.com/Melenium2/kademlia/internal/kbuckets"
	"github.com/Melenium2/kademlia/internal/node"

	"github.com/Melenium2/kademlia/pkg/logger"
)

const LookupResultLimit = 3

// Finder is interface which provide functionality to network find possibility.
// Finder try to connect to Node with settings provided by the arguments
// and return the closest nodes to provided Node.
type finder interface {
	FindNode(id *node.Node, distances []uint) ([]*node.Node, error)
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
	// we need save pointer to self node, because we need to calculate distances
	// between self node and target node, while scan.
	selfID node.ID

	// nodes which already asked for new closer nodes.
	askedNodes map[node.ID]struct{}
	// nodes which was already processed and stored in resultNodes.
	seenNodes map[node.ID]struct{}
	// result of lookup request. This structure will sort nodes
	// by their distance between self node and other nodes.
	resultNodes kbuckets.OrderedNodes
	// nodes for initializing lookup process. At least one node needed for
	// starting process.
	bootstrap []*node.Node
	// count of concurrent searches. If started equals to zero this is
	// means that we already looked at each node in network.
	started int
}

// newLookup create new lookup mechanism.
func newLookup(finder finder, self *node.Node, cfg lookupConfig) *lookup {
	return &lookup{
		finder:      finder,
		log:         logger.GetLogger(),
		selfID:      self.ID(),
		askedNodes:  make(map[node.ID]struct{}),
		seenNodes:   make(map[node.ID]struct{}),
		resultNodes: kbuckets.NewOrderedNodes(self, BucketSize),
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

		if l.selfID == n.ID() {
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
		nodes := l.resultNodes.Nodes()
		// we loop over all nodes and query it for closest nodes. We can not
		// run more parallel scans than Alpha (3).
		for i := 0; i < len(nodes) && l.started < Alpha; i++ {
			curr := nodes[i]

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
	distance := node.DistancesBetween(l.selfID, curr.ID(), LookupResultLimit)

	nodes, err := l.finder.FindNode(curr, distance)
	if err != nil {
		errCh <- err

		return
	}

	resCh <- nodes
}
