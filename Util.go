package main

import (
	"crypto/rand"
)

func generateRandomAscii(size uint) []byte {
	output := make([]byte, size)

	rand.Read(output)

	for i, _ := range output {
		output[i] = 'a' + (output[i] % 26)
	}

	return output
}

type NullWriter struct{}

func (w *NullWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
