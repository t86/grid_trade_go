package debugview

import (
	"time"

	"grid_trade/internal/domain"
	"grid_trade/internal/gateway"
)

type Severity string
type AlertCategory string
type Health string
type AlertStatus string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

const (
	CategoryConnection AlertCategory = "connection"
	CategoryProtection AlertCategory = "protection"
)

const (
	HealthHealthy  Health = "healthy"
	HealthWarning  Health = "warning"
	HealthDegraded Health = "degraded"
)

const (
	AlertActive    AlertStatus = "active"
	AlertRecovered AlertStatus = "recovered"
)

type AlertSummary struct {
	ID       string            `json:"id"`
	Severity Severity          `json:"severity"`
	Category AlertCategory     `json:"category"`
	Market   domain.MarketType `json:"market"`
	Account  string            `json:"account"`
	Title    string            `json:"title"`
	Detail   string            `json:"detail"`
	Since    time.Time         `json:"since"`
	Status   AlertStatus       `json:"status"`
}

type MarketHealthCard struct {
	Market                   domain.MarketType `json:"market"`
	Health                   Health            `json:"health"`
	OnlineAccounts           int               `json:"onlineAccounts"`
	DegradedAccounts         int               `json:"degradedAccounts"`
	AvgHeartbeatLagMs        int64             `json:"avgHeartbeatLagMs"`
	LastReconnectAt          *time.Time        `json:"lastReconnectAt,omitempty"`
	ReduceOnlyAccounts       int               `json:"reduceOnlyAccounts"`
	ListenKeyHealthyAccounts int               `json:"listenKeyHealthyAccounts"`
	LastError                string            `json:"lastError"`
}

type AccountConnectionRow struct {
	Account            string               `json:"account"`
	SessionState       gateway.SessionState `json:"sessionState"`
	UserStreamState    string               `json:"userStreamState"`
	ListenKeyState     string               `json:"listenKeyState"`
	ListenKeyExpiresAt *time.Time           `json:"listenKeyExpiresAt,omitempty"`
	LastHeartbeatAt    *time.Time           `json:"lastHeartbeatAt,omitempty"`
	LastReconnectAt    *time.Time           `json:"lastReconnectAt,omitempty"`
	ReduceOnly         bool                 `json:"reduceOnly"`
	KillSwitch         bool                 `json:"killSwitch"`
	ActiveBackoff      bool                 `json:"activeBackoff"`
	LastError          string               `json:"lastError"`
}

type Dashboard struct {
	Alerts  []AlertSummary     `json:"alerts"`
	Markets []MarketHealthCard `json:"markets"`
}

type Event struct {
	Message string `json:"message"`
}
