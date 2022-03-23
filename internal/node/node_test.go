package node_test

import (
	"testing"

	"github.com/Melenium2/kademlia/internal/node"

	"github.com/stretchr/testify/assert"
)

var (
	id1 = node.ID{
		0x6F, 0xF7, 0x54, 0x41, 0xE2, 0x6D, 0x9D, 0xE0, 0xEA, 0x9A,
		0xA7, 0x06, 0xBA, 0x14, 0x95, 0xCF, 0xBE, 0xB3, 0xD7, 0x87,
	}
	id2 = node.ID{
		0x47, 0xB3, 0x57, 0x03, 0x06, 0x9E, 0xFC, 0xC6, 0xC6, 0xF3,
		0xAC, 0x28, 0x06, 0x52, 0x32, 0xDF, 0x0A, 0x3B, 0xD9, 0x17,
	}
	id3 = node.ID{
		0x03, 0xD0, 0xE3, 0x2E, 0x96, 0x30, 0x10, 0x96, 0x64, 0xC8,
		0x2E, 0x49, 0xA6, 0x7F, 0x80, 0x15, 0x25, 0x08, 0x17, 0x78,
	}
)

func TestDistanceCmp(t *testing.T) {
	var tt = []struct {
		name     string
		a        node.ID
		b        node.ID
		target   node.ID
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

func TestLogDistance(t *testing.T) {
	var tt = []struct {
		name     string
		a        node.ID
		b        node.ID
		expected int
	}{
		{
			name:     "should return 158",
			a:        id1,
			b:        id2,
			expected: 158,
		},
		{
			name:     "should return 159",
			a:        id2,
			b:        id3,
			expected: 159,
		},
	}

	t.Parallel()

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			distance := node.LogDistance(tc.a, tc.b)
			assert.Equal(t, tc.expected, distance)
		})
	}
}

func TestDistancesBetween(t *testing.T) {
	var tt = []struct {
		name     string
		a        node.ID
		b        node.ID
		limit    int
		expected []uint
	}{
		{
			name:     "should return slice with 158 and 158+- 1",
			a:        id1,
			b:        id2,
			limit:    3,
			expected: []uint{158, 159, 157},
		},
		{
			name:     "should return slice with 158 +- 1 and +-2 [158, 159, 157, 160, 156]",
			a:        id1,
			b:        id2,
			limit:    5,
			expected: []uint{158, 159, 157, 160, 156},
		},
		{
			name:     "should return only LogDistance between provided nodes",
			a:        id1,
			b:        id2,
			limit:    1,
			expected: []uint{158},
		},
		{
			name:     "if limit is 0 then return only LogDistance between nodes",
			a:        id1,
			b:        id2,
			limit:    0,
			expected: []uint{158},
		},
	}

	t.Parallel()

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			res := node.DistancesBetween(tc.a, tc.b, tc.limit)
			assert.Equal(t, tc.expected, res)
		})
	}
}
