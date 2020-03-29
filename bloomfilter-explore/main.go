package main

// Adapted from https://codeburst.io/lets-implement-a-bloom-filter-in-go-b2da8a4b849f

import (
	"fmt"
	"hash"
	"hash/fnv"

	"github.com/spaolacci/murmur3"
)

type IBloomFilter interface {
	Add(item []byte)       // Adds the item into the Set
	Test(item []byte) bool // Check if items is maybe in the Set
}

// BloomFilter probabilistic data structure definition
type BloomFilter struct {
	bitset    []bool        // The bloom-filter bitset
	k         uint          // Number of hash values
	n         uint          // Number of elements in the filter
	m         uint          // Size of the bloom filter bitset
	hashFuncs []hash.Hash64 // The hash functions
}

// New returns a ptr to a BloomFilter
func New(size uint) *BloomFilter {
	return &BloomFilter{
		bitset: make([]bool, size),
		k:      3, // we have 3 hash functions for now
		m:      size,
		n:      0,
		hashFuncs: []hash.Hash64{
			murmur3.New64(),
			fnv.New64(),
			fnv.New64a(),
		},
	}
}

// Add the item into the bloom filter set by hashing in over the . // hash functions
func (bf *BloomFilter) Add(item []byte) {
	hashes := bf.hashValues(item)
	for i := uint(0); i < uint(bf.k); i++ {
		position := uint(hashes[i]) % bf.m
		bf.bitset[uint(position)] = true
	}
	bf.n++
}

// hashValues calculates all the hash values by applying in the item over the // hash functions
func (bf *BloomFilter) hashValues(item []byte) []uint64 {
	var result []uint64
	for _, hashFunc := range bf.hashFuncs {
		hashFunc.Write(item)
		result = append(result, hashFunc.Sum64())
		hashFunc.Reset()
	}
	return result
}

// TestIsPresent tests if the item in the bloom filter is set by hashing in over // the hash functions
func (bf *BloomFilter) TestIsPresent(item []byte) (exists bool) {
	hashes := bf.hashValues(item)
	exists = true
	for i := uint(0); i < uint(bf.k); i++ {
		position := uint(hashes[i]) % bf.m
		if !bf.bitset[uint(position)] {
			exists = false
			break
		}
	}
	return
}

func main() {
	bf := New(20)
	for i := 0; i < 20; i++ {
		bf.Add([]byte("hello"))
	}
	fmt.Println(bf.TestIsPresent([]byte("hello")))
	fmt.Println(bf.TestIsPresent([]byte("world")))
	fmt.Println(bf.TestIsPresent([]byte("hi")))
}
