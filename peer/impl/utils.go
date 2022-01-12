package impl

import (
	"math/rand"
	"sync"
)

// TODO: add note, we need to seperate it, because neighbors might be changed across
// different call of neighs. this time I did not consider well on the possible inconsistency and race
func budgetAllocation(neis_ []string, budget uint) ([]string, []uint) {
	if len(neis_) == 0 || budget == 0 {
		return []string{}, []uint{}
	}

	neis := make([]string, len(neis_))
	copy(neis, neis_)
	budgetPerNei := int(budget) / len(neis)
	if budgetPerNei < 1 {
		rand.Shuffle(int(budget), func(i, j int) {
			tmp := neis[i]
			neis[i] = neis[j]
			neis[j] = tmp
		})
		budgets := make([]uint, budget)
		for i := 0; i < int(budget); i++ {
			budgets[i] = 1
		}
		return neis[:int(budget)], budgets
	} else {
		luckyNei := neis[rand.Int31n(int32(len(neis)))]
		luckyBudget := int(budget) - budgetPerNei*(len(neis)-1)
		budgets := make([]uint, len(neis))
		for i := range neis {
			if neis[i] == luckyNei {
				budgets[i] = uint(luckyBudget)
			} else {
				budgets[i] = uint(budgetPerNei)
			}
		}
		return neis, budgets
	}
}

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