// Code generated by mockery v2.10.0. DO NOT EDIT.

package mocks

import (
	"github.com/Melenium2/kademlia/internal/kbuckets"
	"github.com/Melenium2/kademlia/internal/node"
	mock "github.com/stretchr/testify/mock"
)

// KBuckets is an autogenerated mock type for the KBuckets type
type KBuckets struct {
	mock.Mock
}

// BucketAtDistance provides a mock function with given fields: dist
func (_m *KBuckets) BucketAtDistance(dist int) *kbuckets.Bucket {
	ret := _m.Called(dist)

	var r0 *kbuckets.Bucket
	if rf, ok := ret.Get(0).(func(int) *kbuckets.Bucket); ok {
		r0 = rf(dist)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*kbuckets.Bucket)
		}
	}

	return r0
}

// WhoAmI provides a mock function with given fields:
func (_m *KBuckets) WhoAmI() *node.Node {
	ret := _m.Called()

	var r0 *node.Node
	if rf, ok := ret.Get(0).(func() *node.Node); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*node.Node)
		}
	}

	return r0
}
