package scheduler

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type ConsistentHash struct {
	hashRing     map[int]string // Hash ring
	sortedHashes []int          // Sorted hashes
	virtualNode  int            // Virtual node
}

// NewConsistentHash creates a ConsistentHash instance.
func NewConsistentHash(virtualNode int) *ConsistentHash {
	hash := &ConsistentHash{
		hashRing:     make(map[int]string),
		sortedHashes: make([]int, 0),
		virtualNode:  virtualNode,
	}

	return hash
}

// Add adds a node to the hash ring.
func (c *ConsistentHash) Add(node string) {
	for i := 0; i < c.virtualNode; i++ {
		hash := int(crc32.ChecksumIEEE([]byte(strconv.Itoa(i) + node)))
		c.hashRing[hash] = node
		c.sortedHashes = append(c.sortedHashes, hash)
	}

	sort.Ints(c.sortedHashes)
}

// Remove removes a node from the hash ring.
func (c *ConsistentHash) Remove(node string) {
	for i := 0; i < c.virtualNode; i++ {
		hash := int(crc32.ChecksumIEEE([]byte(strconv.Itoa(i) + node)))
		delete(c.hashRing, hash)
	}

	c.rebalance()
}

// Get returns the closest node in the hash ring.
func (c *ConsistentHash) Get(key string) string {
	if len(c.hashRing) == 0 {
		return ""
	}

	hash := int(crc32.ChecksumIEEE([]byte(key)))
	idx := sort.Search(len(c.sortedHashes), func(i int) bool {
		return c.sortedHashes[i] >= hash
	})

	if idx == len(c.sortedHashes) {
		idx = 0
	}

	return c.hashRing[c.sortedHashes[idx]]
}

// rebalance rebalances the hash ring.
func (c *ConsistentHash) rebalance() {
	newHashRing := make(map[int]string)
	newSortedHashes := make([]int, 0, len(c.hashRing))

	for hash, node := range c.hashRing {
		newHashRing[hash] = node
		newSortedHashes = append(newSortedHashes, hash)
	}

	sort.Ints(newSortedHashes)

	c.hashRing = newHashRing
	c.sortedHashes = newSortedHashes
}
