package chord

import "sync"

type MutexString struct {
	sync.Mutex
	data string
}

func (m *MutexString) read() string {
	m.Lock()
	data := m.data
	m.Unlock()
	return data
}

func (m *MutexString) write(data string) {
	m.Lock()
	m.data = data
	m.Unlock()
}

func betweenRightInclude(id uint, left uint, right uint) bool {
	return between(id, left, right) || id == right
}

func between(id uint, left uint, right uint) bool {
	if right > left {
		return id > left && id < right
	} else if right < left {
		return id < right || id > left
	} else {
		return false
	}
}