package capacitymanager

import (
	"time"

	sync "github.com/lukemarsden/golang-mutex-tracer"
)

type ItemList struct {
	items []CapacityManagerItem
	mu    sync.Mutex
}

func NewItemList() *ItemList {
	i := &ItemList{
		items: []CapacityManagerItem{},
	}
	i.mu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "ItemList.mu",
	})
	return i
}

func (l *ItemList) Add(item CapacityManagerItem) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = append(l.items, item)
}

func (l *ItemList) Remove(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	newArr := []CapacityManagerItem{}
	for _, i := range l.items {
		if i.ID != id {
			newArr = append(newArr, i)
		}
	}
	l.items = newArr
}

func (l *ItemList) Get(id string) *CapacityManagerItem {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, item := range l.items {
		if item.ID == id {
			return &item
		}
	}
	return nil
}

func (l *ItemList) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.items)
}

func (l *ItemList) Iterate(handler func(item CapacityManagerItem)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, item := range l.items {
		handler(item)
	}
}
