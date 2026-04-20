package state

import (
	"sync"

	"grid_trade/internal/domain"
)

type Store struct {
	mu        sync.RWMutex
	orders    map[string]domain.OrderRecord
	positions map[domain.PositionKey]domain.PositionSnapshot
	balances  map[string]domain.Balance
}

func NewStore() *Store {
	return &Store{
		orders:    make(map[string]domain.OrderRecord),
		positions: make(map[domain.PositionKey]domain.PositionSnapshot),
		balances:  make(map[string]domain.Balance),
	}
}

func (s *Store) UpsertOrder(order domain.OrderRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders[order.ClientOrderID] = order
}

func (s *Store) Order(clientOrderID string) (domain.OrderRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.orders[clientOrderID]
	return order, ok
}
