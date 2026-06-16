package api

import "sync"

type MutexMap struct {
	values map[string]*sync.Mutex
}

// NewMutexMap creates a new ready to use MutexMap.
func NewMutexMap() *MutexMap {
	return &MutexMap{
		values: make(map[string]*sync.Mutex),
	}
}

// Get retrieves the mutex of k.
//
// If k does not exist, then k will be added to the map.
func (m *MutexMap) Get(k string) *sync.Mutex {
	mutex, ok := m.values[k]
	if !ok {
		mutex = &sync.Mutex{}
		m.values[k] = mutex
	}

	return mutex
}

// Remove removes k from the map.
//
// If k does not exist, then this will do nothing.
func (m *MutexMap) Remove(k string) {
	delete(m.values, k)
}
