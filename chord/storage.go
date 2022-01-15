package chord

import "sync"

type ChordStorage struct {
	sync.Mutex
	data map[uint]uint
}

func (c *ChordStorage) get(key uint) (uint, bool) {
	c.Lock()
	defer c.Unlock()
	data, ok := c.data[key]
	return data, ok
}

func (c *ChordStorage) put(key uint, data uint) {
	c.Lock()
	defer c.Unlock()
	c.data[key] = data
}

