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
			Account:            account.Account,
			SessionState:       account.SessionState,
			SecretRef:          account.SecretRef,
			SecretStatus:       account.SecretStatus,
			UserStreamState:    account.UserStreamState,
			ListenKeyState:     account.ListenKeyState,
			ListenKeyExpiresAt: account.ListenKeyExpiresAt,
			LastHeartbeatAt:    account.LastHeartbeatAt,
			LastReconnectAt:    account.LastReconnectAt,
			ReduceOnly:         account.ReduceOnly,
			KillSwitch:         false,
			ActiveBackoff:      account.ActiveBackoff,
			LastError:          account.LastError,
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
	snapshot := s.gateway.Snapshot()
	events := make([]Event, 0, len(snapshot.Events))
	for _, event := range snapshot.Events {
		events = append(events, Event{
			Message: event.Message,
		})
	}
	return events
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
				Title:    "用户流降级",
				Detail:   account.LastError,
				Since:    now,
				Status:   AlertActive,
			})
		}
		if account.SecretStatus == "load_failed" {
			alerts = append(alerts, AlertSummary{
				ID:       fmt.Sprintf("%s:%s:secret", account.Account, account.Market),
				Severity: SeverityCritical,
				Category: CategoryConnection,
				Market:   account.Market,
				Account:  account.Account,
				Title:    "密钥加载失败",
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
				Title:    "只减仓模式已开启",
				Detail:   "当前交易被限制为只减仓模式",
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
		var heartbeatTotal int64
		var heartbeatCount int64
		for _, account := range accounts {
			if account.Market != market {
				continue
			}
			if account.SessionState == gateway.StateActive {
				card.OnlineAccounts++
			}
			if account.ListenKeyState == "healthy" {
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
			if account.HeartbeatLagMs > 0 {
				heartbeatTotal += account.HeartbeatLagMs
				heartbeatCount++
			}
			if card.LastReconnectAt == nil && account.LastReconnectAt != nil {
				card.LastReconnectAt = account.LastReconnectAt
			}
		}
		if heartbeatCount > 0 {
			card.AvgHeartbeatLagMs = heartbeatTotal / heartbeatCount
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
