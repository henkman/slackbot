package main

import (
	"crypto/rand"
	"log"
	"math/big"
)

func RandUint32(n uint32) uint32 {
	var bn big.Int
	bn.SetUint64(uint64(n))
	r, err := rand.Int(rand.Reader, &bn)
	if err != nil {
		log.Println(err)
		return 0
	}
	return uint32(r.Uint64())
}
