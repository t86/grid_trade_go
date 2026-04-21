package gateway

import (
	"context"
	"io"
	"sort"
	"sync"
	"time"

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

type Snapshot struct {
	Accounts []AccountSnapshot
	Events   []EventSnapshot
}

type AccountSnapshot struct {
	Account            string
	Market             domain.MarketType
	SessionState       SessionState
	UserStreamState    string
	ListenKeyState     string
	ListenKeyExpiresAt *time.Time
	LastHeartbeatAt    *time.Time
	HeartbeatLagMs     int64
	LastReconnectAt    *time.Time
	ReduceOnly         bool
	ActiveBackoff      bool
	LastError          string
}

type EventSnapshot struct {
	Account   string            `json:"account"`
	Market    domain.MarketType `json:"market"`
	Category  string            `json:"category"`
	Message   string            `json:"message"`
	Timestamp time.Time         `json:"timestamp"`
}

type Service struct {
	mu              sync.RWMutex
	sessions        map[string]SessionState
	markets         map[accountMarketKey]SessionState
	userStreams     map[accountMarketKey]string
	listenKeyStates map[accountMarketKey]string
	listenKeyExpiry map[accountMarketKey]*time.Time
	lastHeartbeat   map[accountMarketKey]*time.Time
	lastReconnect   map[accountMarketKey]*time.Time
	activeBackoff   map[accountMarketKey]bool
	lastErrors      map[accountMarketKey]string
	conns           map[string][]io.Closer
	events          []EventSnapshot
	reduceOnly      bool
	listenKeys      ListenKeyProvider
	connector       UserStreamConnector
	now             func() time.Time
}

type accountMarketKey struct {
	account string
	market  domain.MarketType
}

type ListenKeyProvider interface {
	CreateListenKey(context.Context, domain.MarketType) (string, error)
}

type UserStreamConnector interface {
	Connect(context.Context, domain.MarketType, string) (io.Closer, error)
}

func NewService(listenKeys ListenKeyProvider, connector UserStreamConnector) *Service {
	return &Service{
		sessions:        make(map[string]SessionState),
		markets:         make(map[accountMarketKey]SessionState),
		userStreams:     make(map[accountMarketKey]string),
		listenKeyStates: make(map[accountMarketKey]string),
		listenKeyExpiry: make(map[accountMarketKey]*time.Time),
		lastHeartbeat:   make(map[accountMarketKey]*time.Time),
		lastReconnect:   make(map[accountMarketKey]*time.Time),
		activeBackoff:   make(map[accountMarketKey]bool),
		lastErrors:      make(map[accountMarketKey]string),
		conns:           make(map[string][]io.Closer),
		listenKeys:      listenKeys,
		connector:       connector,
		now:             time.Now,
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
	for _, market := range markets {
		key := accountMarketKey{account: account, market: market}
		s.markets[key] = StateConnecting
		s.userStreams[key] = "connecting"
		s.listenKeyStates[key] = "creating"
	}
	s.mu.Unlock()

	var closers []io.Closer
	for _, market := range markets {
		listenKey, err := s.listenKeys.CreateListenKey(ctx, market)
		if err != nil {
			s.markMarketDegraded(account, market, err.Error())
			s.MarkUserStreamDown(account)
			s.closeAll(closers)
			return err
		}

		conn, err := s.connector.Connect(ctx, market, listenKey)
		if err != nil {
			s.markMarketDegraded(account, market, err.Error())
			s.MarkUserStreamDown(account)
			s.closeAll(closers)
			return err
		}
		closers = append(closers, conn)
	}

	s.mu.Lock()
	s.sessions[account] = StateActive
	for _, market := range markets {
		key := accountMarketKey{account: account, market: market}
		s.markets[key] = StateActive
		s.userStreams[key] = "connected"
		s.listenKeyStates[key] = "healthy"
		now := s.now()
		s.lastReconnect[key] = &now
		delete(s.lastErrors, key)
		s.recordEventLocked(account, market, "connection", "user stream connected")
	}
	s.conns[account] = closers
	s.reduceOnly = false
	s.mu.Unlock()
	return nil
}

func (s *Service) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accounts := make([]AccountSnapshot, 0, len(s.markets))
	for key, state := range s.markets {
		var heartbeatLagMs int64
		if heartbeat := s.lastHeartbeat[key]; heartbeat != nil {
			heartbeatLagMs = s.now().Sub(*heartbeat).Milliseconds()
		}
		accounts = append(accounts, AccountSnapshot{
			Account:            key.account,
			Market:             key.market,
			SessionState:       state,
			UserStreamState:    s.userStreams[key],
			ListenKeyState:     s.listenKeyStates[key],
			ListenKeyExpiresAt: s.listenKeyExpiry[key],
			LastHeartbeatAt:    s.lastHeartbeat[key],
			HeartbeatLagMs:     heartbeatLagMs,
			LastReconnectAt:    s.lastReconnect[key],
			ReduceOnly:         s.reduceOnly,
			ActiveBackoff:      s.activeBackoff[key],
			LastError:          s.lastErrors[key],
		})
	}
	sort.Slice(accounts, func(i, j int) bool {
		if accounts[i].Market == accounts[j].Market {
			return accounts[i].Account < accounts[j].Account
		}
		return accounts[i].Market < accounts[j].Market
	})

	events := append([]EventSnapshot(nil), s.events...)

	return Snapshot{Accounts: accounts, Events: events}
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

func (s *Service) markMarketDegraded(account string, market domain.MarketType, lastError string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := accountMarketKey{account: account, market: market}
	s.markets[key] = StateDegraded
	s.userStreams[key] = "disconnected"
	s.listenKeyStates[key] = "stale"
	s.lastErrors[key] = lastError
	s.recordEventLocked(account, market, "connection", lastError)
}

func (s *Service) RecordHeartbeat(account string, market domain.MarketType, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := accountMarketKey{account: account, market: market}
	s.lastHeartbeat[key] = &at
	s.userStreams[key] = "connected"
	s.recordEventLocked(account, market, "connection", "heartbeat received")
}

func (s *Service) MarkListenKeyState(account string, market domain.MarketType, state string, expiresAt *time.Time, detail string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := accountMarketKey{account: account, market: market}
	s.listenKeyStates[key] = state
	s.listenKeyExpiry[key] = expiresAt
	if detail != "" {
		s.lastErrors[key] = detail
	}
	s.recordEventLocked(account, market, "connection", "listen key "+state)
}

func (s *Service) MarkBackoff(account string, market domain.MarketType, active bool, detail string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := accountMarketKey{account: account, market: market}
	s.activeBackoff[key] = active
	if active {
		s.markets[key] = StateBackoff
	} else if s.markets[key] == StateBackoff {
		s.markets[key] = StateActive
	}
	if detail != "" {
		s.lastErrors[key] = detail
	}
	if active {
		s.recordEventLocked(account, market, "protection", "backoff enabled")
	} else {
		s.recordEventLocked(account, market, "protection", "backoff cleared")
	}
}

func (s *Service) recordEventLocked(account string, market domain.MarketType, category, message string) {
	s.events = append(s.events, EventSnapshot{
		Account:   account,
		Market:    market,
		Category:  category,
		Message:   message,
		Timestamp: s.now(),
	})
	if len(s.events) > 100 {
		s.events = append([]EventSnapshot(nil), s.events[len(s.events)-100:]...)
	}
}
