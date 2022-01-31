package crypto

import (
	"crypto/rand"
	"crypto/sha1"
)

func Sha1() ([]byte, error) {
	hash := sha1.New()

	str := make([]byte, 100) // nolint:gomnd

	_, err := rand.Read(str)
	if err != nil {
		return nil, err
	}

	_, err = hash.Write(str)
	if err != nil {
		return nil, err
	}

	sum := hash.Sum(nil)

	return sum, nil
}
