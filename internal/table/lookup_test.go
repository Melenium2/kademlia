package table

import (
	"sort"
	"testing"

	"github.com/Melenium2/kademlia"
	"github.com/Melenium2/kademlia/internal/table/mocks"
	"github.com/Melenium2/kademlia/internal/table/node"

	"github.com/stretchr/testify/assert"
)

func randomNode() *node.Node {
	newNode, _ := kademlia.NewNode()

	return node.WrapNode(newNode)
}

func TestScan_Should_call_Find_func_and_return_result_to_result_channel(t *testing.T) {
	var (
		targetNode = randomNode()
		resCh      = make(chan []*node.Node, Alpha)
		errCh      = make(chan error, 1)
		expected   = []*node.Node{
			randomNode(), randomNode(), randomNode(), randomNode(),
		}
	)

	fakeFinder := mocks.Finder{}
	fakeFinder.On("Find", targetNode.ID(), targetNode.IP()).Return(expected, nil)

	l := newLookup(lookupConfig{}, &fakeFinder, nil)

	l.scan(targetNode, resCh, errCh)

	select {
	case nodes := <-resCh:
		assert.Equal(t, expected, nodes)
	case <-errCh:
		// nothing here
	}
}

func TestScan_Should_call_Find_func_which_return_some_error(t *testing.T) {
	var (
		targetNode = randomNode()
		resCh      = make(chan []*node.Node, Alpha)
		errCh      = make(chan error, 1)
	)

	fakeFinder := mocks.Finder{}
	fakeFinder.On("Find", targetNode.ID(), targetNode.IP()).Return(nil, ErrEmptyBootstrapNodes)

	l := newLookup(lookupConfig{}, &fakeFinder, nil)

	l.scan(targetNode, resCh, errCh)

	select {
	case <-resCh:
	// nothing here
	case err := <-errCh:
		assert.Error(t, err)
		assert.Equal(t, ErrEmptyBootstrapNodes, err)
	}
}

func TestConsume_Should_save_all_incoming_nodes_from_provided_channel_also_nodes_should_be_sorted_by_id(t *testing.T) {
	var (
		self     = randomNode()
		expected = []*node.Node{randomNode(), randomNode(), randomNode()}
		resCh    = make(chan []*node.Node, Alpha)
		errCh    = make(chan error, 1)
	)
	l := newLookup(lookupConfig{}, nil, self)

	resCh <- expected

	err := l.consume(resCh, errCh)
	assert.NoError(t, err)

	for i := 0; i < len(expected); i++ {
		_, ok := l.seenNodes[expected[i].ID()]
		assert.True(t, ok)
	}

	assert.Equal(t, len(expected), len(l.resultNodes.nodes))

	sort.Slice(expected, func(i, j int) bool {
		return node.DistanceCmp(self.ID(), expected[i].ID(), expected[j].ID()) < 0
	})

	assert.Equal(t, expected, l.resultNodes.Nodes())
}
