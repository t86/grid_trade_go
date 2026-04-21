package gateway

import (
	"context"
	"io"
	"sync"

	"grid_trade/internal/domain"
)

type SessionState string

const (
	StateConnecting SessionState = "connecting"
	StateActive     SessionState = "active"
	StateDegraded   SessionState = "degraded"
	StateBackoff    SessionState = "backing_off"
	StateBanned     SessionState = "banned_until"
)

type Health struct {
	TradingEnabled bool
	ReduceOnly     bool
	Sessions       map[string]SessionState
}

type Service struct {
	mu         sync.RWMutex
	sessions   map[string]SessionState
	conns      map[string][]io.Closer
	reduceOnly bool
	listenKeys ListenKeyProvider
	connector  UserStreamConnector
}

type ListenKeyProvider interface {
	CreateListenKey(context.Context, domain.MarketType) (string, error)
}

type UserStreamConnector interface {
	Connect(context.Context, domain.MarketType, string) (io.Closer, error)
}

func NewService(listenKeys ListenKeyProvider, connector UserStreamConnector) *Service {
	return &Service{
		sessions:   make(map[string]SessionState),
		conns:      make(map[string][]io.Closer),
		listenKeys: listenKeys,
		connector:  connector,
	}
}

func (s *Service) MarkUserStreamDown(account string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[account] = StateDegraded
	s.reduceOnly = true
}

func (s *Service) BootstrapAccount(ctx context.Context, account string, markets []domain.MarketType) error {
	s.mu.Lock()
	s.sessions[account] = StateConnecting
	s.mu.Unlock()

	var closers []io.Closer
	for _, market := range markets {
		listenKey, err := s.listenKeys.CreateListenKey(ctx, market)
		if err != nil {
			s.MarkUserStreamDown(account)
			s.closeAll(closers)
			return err
		}

		conn, err := s.connector.Connect(ctx, market, listenKey)
		if err != nil {
			s.MarkUserStreamDown(account)
			s.closeAll(closers)
			return err
		}
		closers = append(closers, conn)
	}

	s.mu.Lock()
	s.sessions[account] = StateActive
	s.conns[account] = closers
	s.reduceOnly = false
	s.mu.Unlock()
	return nil
}

func (s *Service) State(account string) SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[account]
}

func (s *Service) Health() Health {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make(map[string]SessionState, len(s.sessions))
	for account, state := range s.sessions {
		sessions[account] = state
	}

	return Health{
		TradingEnabled: !s.reduceOnly,
		ReduceOnly:     s.reduceOnly,
		Sessions:       sessions,
	}
}

func (s *Service) closeAll(closers []io.Closer) {
	for _, closer := range closers {
		_ = closer.Close()
	}
}
