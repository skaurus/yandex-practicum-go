package utils

import "math/rand"

const asciiSymbols = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// RandStringN - see https://stackoverflow.com/a/31832326/320345
func RandStringN(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = asciiSymbols[rand.Int63()%int64(len(asciiSymbols))]
	}
	return string(b)
}
