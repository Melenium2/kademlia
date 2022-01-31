package node_test

import (
	"testing"

	"github.com/Melenium2/kademlia"
	"github.com/Melenium2/kademlia/internal/table/node"
	"github.com/stretchr/testify/assert"
)

var (
	id1 = kademlia.ID{
		0x6F, 0xF7, 0x54, 0x41, 0xE2, 0x6D, 0x9D, 0xE0, 0xEA, 0x9A,
		0xA7, 0x06, 0xBA, 0x14, 0x95, 0xCF, 0xBE, 0xB3, 0xD7, 0x87,
	}
	id2 = kademlia.ID{
		0x47, 0xB3, 0x57, 0x03, 0x06, 0x9E, 0xFC, 0xC6, 0xC6, 0xF3,
		0xAC, 0x28, 0x06, 0x52, 0x32, 0xDF, 0x0A, 0x3B, 0xD9, 0x17,
	}
	id3 = kademlia.ID{
		0x03, 0xD0, 0xE3, 0x2E, 0x96, 0x30, 0x10, 0x96, 0x64, 0xC8,
		0x2E, 0x49, 0xA6, 0x7F, 0x80, 0x15, 0x25, 0x08, 0x17, 0x78,
	}
)

func TestDistanceCmp(t *testing.T) {
	var tt = []struct {
		name     string
		a        kademlia.ID
		b        kademlia.ID
		target   kademlia.ID
		expected int
	}{
		{
			name:     "should return 0 because distance between self-a == self-b",
			a:        id1,
			b:        id1,
			target:   id2,
			expected: 0,
		},
		{
			name:     "should return 1 because distance between self-a > self-b",
			a:        id1,
			b:        id2,
			target:   id3,
			expected: 1,
		},
		{
			name:     "should return -1 because distance between self-a < self-b",
			a:        id2,
			b:        id1,
			target:   id3,
			expected: -1,
		},
	}

	t.Parallel()

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cmp := node.DistanceCmp(tc.target, tc.a, tc.b)
			assert.Equal(t, tc.expected, cmp)
		})
	}
}
