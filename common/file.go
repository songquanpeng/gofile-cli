package common

import (
	"crypto/sha256"
	"io"
	"log"
	"os"
)

func CalculateChecksumAndSize(filename string) ([]byte, uint32) {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}
	buffer := make([]byte, 0)
	fi, err := f.Stat()
	if err != nil {
		log.Fatal(err)
	}
	return h.Sum(buffer), uint32(fi.Size())
}
