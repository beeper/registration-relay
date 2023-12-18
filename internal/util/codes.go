package util

import (
	"crypto/rand"
	"math/big"
)

const codeLetters = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

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
	randBytes, err := generateRandomBytes(16)
	if err != nil {
		return "", err
	}
	return string(randBytes[:4]) + "-" + string(randBytes[4:8]) + "-" + string(randBytes[8:12]) + "-" + string(randBytes[12:16]), nil
}
