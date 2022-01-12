package impl

import (
	"fmt"
	"go.dedis.ch/cs438/peer"
	"sync"
)


func NewFingerTable(conf peer.Configuration) *FingerTable {
	fingerTable := FingerTable{}
	fingerTable.data = make([]string, conf.ChordBits)
	fingerTable.fixPointer = 0
	return &fingerTable
}

type FingerTable struct {
	sync.Mutex
	data []string
	fixPointer uint
}

func (f *FingerTable) insert(loc int, address string) error {
	if loc < len(f.data) && loc >= 0 {
		f.Lock()
		f.data[loc] = address
		f.Unlock()
		return nil
	} else {
		return fmt.Errorf("Insert: insert finger table is out of range! ")
	}
}

func (f *FingerTable) load(loc int) (string, error) {
	if loc < len(f.data) && loc >= 0 {
		f.Lock()
		address := f.data[loc]
		f.Unlock()
		return address, nil
	} else {
		return "", fmt.Errorf("Load: the index of finger table is out of range! ")
	}
}