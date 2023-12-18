package util

import (
	"crypto/rand"
	"math/big"
)

const codeLetters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func generateRandomBytes(n int) ([]byte, error) {
	max := big.NewInt(int64(len(codeLetters)))

	buf := make([]byte, n)

	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, max)
		if err != nil {
			return nil, err
		}
		buf[i] = codeLetters[num.Int64()]
	}

	return buf, nil
}

func GenerateProviderCode() (string, error) {
	randBytes, err := generateRandomBytes(9)
	if err != nil {
		return "", err
	}
	return string(randBytes[:3]) + "-" + string(randBytes[3:6]) + "-" + string(randBytes[6:9]), nil
}
