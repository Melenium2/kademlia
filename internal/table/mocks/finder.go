// Code generated by mockery v2.10.0. DO NOT EDIT.

package mocks

import (
	"github.com/Melenium2/kademlia/internal/node"
	mock "github.com/stretchr/testify/mock"
)

// Finder is an autogenerated mock type for the Finder type
type Finder struct {
	mock.Mock
}

// Find provides a mock function with given fields: id, distances
func (_m *Finder) FindNode(id *node.Node, distances []uint) ([]*node.Node, error) {
	ret := _m.Called(id, distances)

	var r0 []*node.Node
	if rf, ok := ret.Get(0).(func(*node.Node, []uint) []*node.Node); ok {
		r0 = rf(id, distances)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*node.Node)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*node.Node, []uint) error); ok {
		r1 = rf(id, distances)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
