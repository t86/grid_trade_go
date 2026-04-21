package gateway

import (
	"context"
	"io"
	"sort"
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

type Snapshot struct {
	Accounts []AccountSnapshot
}

type AccountSnapshot struct {
	Account      string
	Market       domain.MarketType
	SessionState SessionState
	ReduceOnly   bool
	LastError    string
}

type Service struct {
	mu         sync.RWMutex
	sessions   map[string]SessionState
	markets    map[accountMarketKey]SessionState
	lastErrors map[accountMarketKey]string
	conns      map[string][]io.Closer
	reduceOnly bool
	listenKeys ListenKeyProvider
	connector  UserStreamConnector
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
		sessions:   make(map[string]SessionState),
		markets:    make(map[accountMarketKey]SessionState),
		lastErrors: make(map[accountMarketKey]string),
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
	for _, market := range markets {
		s.markets[accountMarketKey{account: account, market: market}] = StateConnecting
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
		delete(s.lastErrors, key)
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
		accounts = append(accounts, AccountSnapshot{
			Account:      key.account,
			Market:       key.market,
			SessionState: state,
			ReduceOnly:   s.reduceOnly,
			LastError:    s.lastErrors[key],
		})
	}
	sort.Slice(accounts, func(i, j int) bool {
		if accounts[i].Market == accounts[j].Market {
			return accounts[i].Account < accounts[j].Account
		}
		return accounts[i].Market < accounts[j].Market
	})

	return Snapshot{Accounts: accounts}
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
	s.lastErrors[key] = lastError
}
