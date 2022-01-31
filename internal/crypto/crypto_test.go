package crypto_test

import (
	"testing"

	"github.com/Melenium2/kademlia/internal/crypto"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestSha1_Should_generate_different_hashes_each_time(t *testing.T) {
	for i := 0; i < 150; i++ {
		sha1, err := crypto.Sha1()
		require.NoError(t, err)

		sha1Second, err := crypto.Sha1()
		require.NoError(t, err)

		assert.NotEqualf(t, sha1, sha1Second, "%s != %s", sha1, sha1Second)
	}
}
