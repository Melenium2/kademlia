package node

import (
	"math/bits"

	"github.com/Melenium2/kademlia/internal/crypto"
)

// DistanceCmp compares the distance between self-Node and Node-a, also,
// self-Node and Node-b. Function returns -1 if a closer to self-Node, 1 if b
// closer to self Node and 0 if distances are equal.
func DistanceCmp(self, a, b ID) int {
	for i := 0; i < len(self); i++ {
		distA := a[i] ^ self[i]
		distB := b[i] ^ self[i]

		if distA > distB {
			return 1
		} else if distA < distB {
			return -1
		}
	}

	return 0
}

// LogDistance returns the logarithmic distance between a and b, log2(a ^ b).
func LogDistance(a, b ID) int {
	// leading zeros
	lz := 0

	for i := range a {
		x := a[i] ^ b[i]

		if x == 0 {
			lz += 8
		} else {
			lz += bits.LeadingZeros8(x)

			break
		}
	}

	return len(a)*8 - lz
}

// DistancesBetween compute the distance parameter for FIND_NODE call.
// It chooses distances adjacent to LogDistance(target, dest), e.g. for a target
// with LogDistance(target, dest) = 255 the result is [255, 256, 254].
func DistancesBetween(a, b ID, limit int) []uint {
	dist := make([]uint, 0)

	targetDist := LogDistance(a, b)
	dist = append(dist, uint(targetDist))

	for i := 1; len(dist) < limit; i++ {
		if targetDist+i < 256 {
			dist = append(dist, uint(targetDist+i))
		}

		if targetDist > 0 {
			dist = append(dist, uint(targetDist-i))
		}
	}

	return dist
}

func GenerateID() (ID, error) {
	sha1, err := crypto.Sha1()
	if err != nil {
		return ID{}, err
	}

	p := (*ID)(sha1)

	return *p, nil
}
