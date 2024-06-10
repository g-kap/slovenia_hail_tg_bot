package storage

import (
	"context"
	"sync"
)

type InMemStorage struct {
	m  map[int64][]string
	mu *sync.Mutex
}

func (i InMemStorage) GetHailSubscriptions(ctx context.Context, chatID int64) ([]string, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.m[chatID], nil
}

func (i InMemStorage) AddHailSubscription(ctx context.Context, chatID int64, region string) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.m[chatID] = append(i.m[chatID], region)
	return nil
}

func (i InMemStorage) DeleteSubscriptions(ctx context.Context, chatID int64) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	delete(i.m, chatID)
	return nil
}

func (i InMemStorage) GetChatsForRegion(ctx context.Context, region string) ([]int64, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	var res []int64
	for chatID, regions := range i.m {
		for _, r := range regions {
			if r == region {
				res = append(res, chatID)
			}
		}
	}
	return res, nil
}

func NewInMemStorage() InMemStorage {
	return InMemStorage{
		m:  make(map[int64][]string),
		mu: &sync.Mutex{},
	}
}
