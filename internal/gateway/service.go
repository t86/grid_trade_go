package gateway

import "sync"

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
	reduceOnly bool
}

func NewService(_, _ any) *Service {
	return &Service{
		sessions: make(map[string]SessionState),
	}
}

func (s *Service) MarkUserStreamDown(account string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[account] = StateDegraded
	s.reduceOnly = true
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
