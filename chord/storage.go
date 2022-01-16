package chord

import "sync"


type ChordStorage struct {
	sync.Mutex
	data map[uint]interface{}
}

func (c *ChordStorage) get(key uint) (interface{}, bool) {
	c.Lock()
	defer c.Unlock()
	data, ok := c.data[key]
	return data, ok
}

func (c *ChordStorage) put(key uint, data interface{}) {
	c.Lock()
	defer c.Unlock()
	c.data[key] = data
}

func (c *ChordStorage) findBetweenRightInclude(begin uint, end uint) map[uint]interface{} {
	res := map[uint]interface{}{}
	c.Lock()
	defer c.Unlock()
	for key, value := range c.data {
		if betweenRightInclude(key, begin, end) {
			res[key] = value
		}
	}
	return res
}

func (c *ChordStorage) deleteBetweenRightInclude(begin uint, end uint) {
	kv := c.findBetweenRightInclude(begin, end)
	c.Lock()
	defer c.Unlock()
	for key := range kv {
		delete(c.data, key)
	}
}