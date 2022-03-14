package table

import (
	"errors"
	"io"
	"net"
	"sort"
	"testing"

	"github.com/Melenium2/kademlia"
	"github.com/Melenium2/kademlia/internal/table/mocks"
	"github.com/Melenium2/kademlia/internal/table/node"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func randomNode() *node.Node {
	addr := &net.UDPAddr{
		IP:   net.IPv4(1, 1, 1, 1),
		Port: 5222,
	}
	newNode := kademlia.NewNode(addr)

	return node.WrapNode(newNode)
}

var self = randomNode()

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
	fakeFinder.On("FindNode", targetNode, mock.IsType([]uint{})).Return(expected, nil)

	l := newLookup(&fakeFinder, self, lookupConfig{})

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
	fakeFinder.On("FindNode", targetNode, mock.IsType([]uint{})).Return(nil, ErrEmptyBootstrapNodes)

	l := newLookup(&fakeFinder, self, lookupConfig{})

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
		expected = []*node.Node{randomNode(), randomNode(), randomNode()}
		resCh    = make(chan []*node.Node, Alpha)
		errCh    = make(chan error, 1)
	)
	l := newLookup(nil, self, lookupConfig{})
	l.started = 1

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
	assert.Equal(t, 0, l.started)
}

func TestConsume_Should_dont_save_already_processed_nodes(t *testing.T) {
	var (
		expected = []*node.Node{randomNode(), randomNode(), randomNode()}
		resCh    = make(chan []*node.Node, Alpha)
		errCh    = make(chan error, 1)
	)
	l := newLookup(nil, self, lookupConfig{})
	l.started = 1

	for _, curr := range expected {
		l.seenNodes[curr.ID()] = struct{}{}
	}

	resCh <- expected

	err := l.consume(resCh, errCh)
	assert.NoError(t, err)

	assert.Equal(t, len(expected), len(l.seenNodes))
	assert.Equal(t, 0, len(l.resultNodes.Nodes()))
	assert.Equal(t, 0, l.started)
}

func TestConsume_Should_return_error_if_error_sent_to_the_channel(t *testing.T) {
	var (
		resCh       = make(chan []*node.Node, Alpha)
		errCh       = make(chan error, 1)
		expectedErr = errors.New("some error while processing")
	)
	l := newLookup(nil, self, lookupConfig{})
	l.started = 1

	errCh <- expectedErr

	err := l.consume(resCh, errCh)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestStart_Should_startup_with_3_nodes_and_check_it_then_add_3_more_nodes_and_check_it_too_then_return_6_nodes_as_result(t *testing.T) {
	var (
		iteration = 3
		resCh     = make(chan []*node.Node, Alpha)
		errCh     = make(chan error, 1)
		result    = []*node.Node{randomNode(), randomNode(), randomNode()}
	)

	fakeFinder := mocks.Finder{}
	fakeFinder.
		On("FindNode", mock.IsType(self), mock.IsType([]uint{})).
		Return(result, nil).
		Times(6)

	l := newLookup(&fakeFinder, self, lookupConfig{})

	for i := 0; i < iteration; i++ {
		l.resultNodes.Add(randomNode())
	}

	err := l.start(resCh, errCh)
	require.NoError(t, err)

	assert.Equal(t, 6, len(l.resultNodes.Nodes()))
	assert.Equal(t, len(l.resultNodes.Nodes()), len(l.askedNodes))

	for _, curr := range l.resultNodes.Nodes() {
		_, ok := l.askedNodes[curr.ID()]
		assert.True(t, ok)
	}
}

func TestStart_Should_return_error_if_can_not_consume_some_nodes(t *testing.T) {
	var (
		iteration   = 3
		resCh       = make(chan []*node.Node, Alpha)
		errCh       = make(chan error, 1)
		expectedErr = errors.New("some expected error")
	)

	fakeFinder := mocks.Finder{}
	fakeFinder.
		On("FindNode", mock.IsType(self), mock.IsType([]uint{})).
		Return(nil, expectedErr)

	l := newLookup(&fakeFinder, self, lookupConfig{})

	for i := 0; i < iteration; i++ {
		l.resultNodes.Add(randomNode())
	}

	err := l.start(resCh, errCh)
	require.Error(t, err)
	assert.Equal(t, err, expectedErr)
}

func TestDrain_Should_drain_all_channels(t *testing.T) {
	var (
		resCh = make(chan []*node.Node, Alpha)
		errCh = make(chan error, 1)
	)

	l := newLookup(nil, self, lookupConfig{})

	resCh <- []*node.Node{randomNode(), randomNode()}
	errCh <- io.ErrClosedPipe

	assert.Equal(t, 1, len(resCh))
	assert.Equal(t, 1, len(errCh))

	close(resCh)
	close(errCh)

	l.drainCh(resCh, errCh)

	assert.Equal(t, 0, len(resCh))
	assert.Equal(t, 0, len(errCh))
}
