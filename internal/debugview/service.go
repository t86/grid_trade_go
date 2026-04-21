package debugview

import (
	"fmt"
	"sort"
	"time"

	"grid_trade/internal/domain"
	"grid_trade/internal/gateway"
)

type GatewaySnapshotter interface {
	Snapshot() gateway.Snapshot
}

type Service struct {
	gateway GatewaySnapshotter
	now     func() time.Time
}

func NewService(gateway GatewaySnapshotter) Service {
	return Service{
		gateway: gateway,
		now:     time.Now,
	}
}

func (s Service) Dashboard() Dashboard {
	snapshot := s.gateway.Snapshot()
	return Dashboard{
		Alerts:  buildAlerts(snapshot.Accounts, s.now()),
		Markets: buildMarketCards(snapshot.Accounts),
	}
}

func (s Service) Accounts(market domain.MarketType) []AccountConnectionRow {
	snapshot := s.gateway.Snapshot()
	rows := make([]AccountConnectionRow, 0)
	for _, account := range snapshot.Accounts {
		if account.Market != market {
			continue
		}
		rows = append(rows, AccountConnectionRow{
			Account:         account.Account,
			SessionState:    account.SessionState,
			UserStreamState: string(account.SessionState),
			ListenKeyState:  listenKeyState(account),
			ReduceOnly:      account.ReduceOnly,
			KillSwitch:      false,
			ActiveBackoff:   account.SessionState == gateway.StateBackoff,
			LastError:       account.LastError,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		li := rowRank(rows[i])
		lj := rowRank(rows[j])
		if li == lj {
			return rows[i].Account < rows[j].Account
		}
		return li < lj
	})
	return rows
}

func (s Service) Events(_ domain.MarketType, _ string) []Event {
	return nil
}

func buildAlerts(accounts []gateway.AccountSnapshot, now time.Time) []AlertSummary {
	alerts := make([]AlertSummary, 0)
	for _, account := range accounts {
		if account.SessionState == gateway.StateDegraded {
			alerts = append(alerts, AlertSummary{
				ID:       fmt.Sprintf("%s:%s:connection", account.Account, account.Market),
				Severity: SeverityCritical,
				Category: CategoryConnection,
				Market:   account.Market,
				Account:  account.Account,
				Title:    "user stream degraded",
				Detail:   account.LastError,
				Since:    now,
				Status:   AlertActive,
			})
		}
		if account.ReduceOnly {
			alerts = append(alerts, AlertSummary{
				ID:       fmt.Sprintf("%s:%s:reduce-only", account.Account, account.Market),
				Severity: SeverityWarning,
				Category: CategoryProtection,
				Market:   account.Market,
				Account:  account.Account,
				Title:    "reduce-only enabled",
				Detail:   "trading is limited to reduce-only mode",
				Since:    now,
				Status:   AlertActive,
			})
		}
	}

	sort.Slice(alerts, func(i, j int) bool {
		if alerts[i].Severity == alerts[j].Severity {
			if alerts[i].Market == alerts[j].Market {
				return alerts[i].Account < alerts[j].Account
			}
			return alerts[i].Market < alerts[j].Market
		}
		return severityRank(alerts[i].Severity) < severityRank(alerts[j].Severity)
	})
	return alerts
}

func buildMarketCards(accounts []gateway.AccountSnapshot) []MarketHealthCard {
	markets := []domain.MarketType{domain.MarketSpot, domain.MarketFuturesUM}
	cards := make([]MarketHealthCard, 0, len(markets))
	for _, market := range markets {
		card := MarketHealthCard{Market: market, Health: HealthHealthy}
		for _, account := range accounts {
			if account.Market != market {
				continue
			}
			if account.SessionState == gateway.StateActive {
				card.OnlineAccounts++
				card.ListenKeyHealthyAccounts++
			}
			if account.SessionState == gateway.StateDegraded {
				card.DegradedAccounts++
				card.Health = HealthDegraded
				if card.LastError == "" && account.LastError != "" {
					card.LastError = account.LastError
				}
			}
			if account.ReduceOnly {
				card.ReduceOnlyAccounts++
				if card.Health == HealthHealthy {
					card.Health = HealthWarning
				}
			}
		}
		cards = append(cards, card)
	}
	return cards
}

func rowRank(row AccountConnectionRow) int {
	switch {
	case row.SessionState == gateway.StateDegraded:
		return 0
	case row.ReduceOnly:
		return 1
	case row.ActiveBackoff:
		return 2
	default:
		return 3
	}
}

func severityRank(severity Severity) int {
	switch severity {
	case SeverityCritical:
		return 0
	case SeverityWarning:
		return 1
	default:
		return 2
	}
}

func listenKeyState(account gateway.AccountSnapshot) string {
	if account.SessionState == gateway.StateActive {
		return "healthy"
	}
	if account.SessionState == gateway.StateDegraded {
		return "stale"
	}
	return "unknown"
}
