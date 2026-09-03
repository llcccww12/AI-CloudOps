package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const queryCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func GenerateReportSecret() string {
	return randomFromAlphabet(16, queryCodeAlphabet+"abcdefghjkmnpqrstuvwxyz")
}

func GenerateQueryCode() string {
	return randomFromAlphabet(6, queryCodeAlphabet)
}

func GenerateReportCode(customerID int) string {
	suffix := randomFromAlphabet(4, queryCodeAlphabet)
	return fmt.Sprintf("CAC-%d-%s", customerID, suffix)
}

func randomFromAlphabet(length int, alphabet string) string {
	if length <= 0 || alphabet == "" {
		return ""
	}
	out := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			out[i] = alphabet[i%len(alphabet)]
			continue
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out)
}
