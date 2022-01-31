package table

import (
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
