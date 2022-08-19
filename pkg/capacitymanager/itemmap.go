package capacitymanager

import (
	"time"

	sync "github.com/lukemarsden/golang-mutex-tracer"
)

type ItemMap struct {
	items map[string]CapacityManagerItem
	mu    sync.Mutex
}

func NewItemMap() *ItemMap {
	i := &ItemMap{
		items: map[string]CapacityManagerItem{},
	}
	i.mu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "ItemMap.mu",
	})
	return i
}

func (m *ItemMap) Add(j CapacityManagerItem) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[j.ID] = j
}

func (m *ItemMap) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, id)
}

func (m *ItemMap) Get(id string) *CapacityManagerItem {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.items[id]
	if !ok {
		return nil
	} else {
		return &item
	}
}

func (m *ItemMap) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.items)
}

func (m *ItemMap) Iterate(handler func(item CapacityManagerItem)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, item := range m.items {
		handler(item)
	}
}
