# Binance WS Trading Service Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建一套基于 Go 的 Binance WebSocket 优先交易服务骨架，支持 Spot、USD-M Futures、双向持仓、统一准入与风控，并接入第一个网格策略插件。

**Architecture:** 服务先以单二进制形式实现，但内部按 `gateway / admission / risk / state / runtime / adapters` 分层。交易链路以 WebSocket 为主，REST 只用于冷启动和恢复对账；策略只产生 `OrderIntent`，不直接触达交易所。

**Tech Stack:** Go 1.24+, standard library, `github.com/gorilla/websocket`, `gopkg.in/yaml.v3`, `github.com/prometheus/client_golang/prometheus`, `github.com/stretchr/testify`

---

## File Structure

本计划会创建或修改以下文件：

- `go.mod`
- `cmd/trader/main.go`
- `internal/config/config.go`
- `internal/config/config_test.go`
- `internal/domain/types.go`
- `internal/domain/types_test.go`
- `internal/kernel/bus/bus.go`
- `internal/kernel/bus/bus_test.go`
- `internal/state/store.go`
- `internal/state/store_test.go`
- `internal/admission/service.go`
- `internal/admission/service_test.go`
- `internal/risk/service.go`
- `internal/risk/service_test.go`
- `internal/limit/limiter.go`
- `internal/limit/limiter_test.go`
- `internal/exchange/binance/common/types.go`
- `internal/exchange/binance/spot/adapter.go`
- `internal/exchange/binance/spot/adapter_test.go`
- `internal/exchange/binance/futures/adapter.go`
- `internal/exchange/binance/futures/adapter_test.go`
- `internal/gateway/service.go`
- `internal/gateway/service_test.go`
- `internal/strategy/runtime/runtime.go`
- `internal/strategy/runtime/runtime_test.go`
- `internal/strategy/grid/strategy.go`
- `internal/strategy/grid/strategy_test.go`
- `internal/logger/logger.go`
- `configs/example.yaml`
- `Makefile`

### Task 1: 初始化仓库和基础工程骨架

**Files:**
- Create: `go.mod`
- Create: `cmd/trader/main.go`
- Create: `internal/logger/logger.go`
- Create: `Makefile`
- Create: `configs/example.yaml`

- [ ] **Step 1: 初始化 Go 模块并建立顶层目录**

```bash
mkdir -p cmd/trader internal/logger configs
go mod init grid_trade
```

Expected: 生成 `go.mod`，模块名为 `grid_trade`。

- [ ] **Step 2: 写最小可运行入口**

```go
package main

import (
    "context"
    "log"
    "os/signal"
    "syscall"
)

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    <-ctx.Done()
    log.Println("trader stopped")
}
```

- [ ] **Step 3: 写基础日志封装和示例配置**

```go
package logger

import (
    "log/slog"
    "os"
)

func New(level slog.Level) *slog.Logger {
    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
    return slog.New(handler)
}
```

```yaml
system:
  log_level: info
  http_addr: ":8080"
accounts:
  - name: primary
    market_types: ["spot", "futures_um"]
    api_key_env: BINANCE_API_KEY
    secret_key_env: BINANCE_SECRET_KEY
strategies:
  - id: grid-btc
    type: grid
    symbol: BTCUSDT
    market: futures_um
```

- [ ] **Step 4: 写基础 Makefile**

```make
test:
	go test ./...

run:
	go run ./cmd/trader

fmt:
	gofmt -w cmd internal
```

- [ ] **Step 5: 运行基础验证**

Run: `go test ./...`  
Expected: PASS，当前只有空包或最小包测试通过。

- [ ] **Step 6: 提交基础工程**

```bash
git init
git add go.mod cmd/trader/main.go internal/logger/logger.go configs/example.yaml Makefile
git commit -m "chore: bootstrap go trading service"
```

Expected: 成功初始化仓库并提交第一版骨架。

### Task 2: 建立统一领域模型、配置加载和事件总线

**Files:**
- Create: `internal/domain/types.go`
- Create: `internal/domain/types_test.go`
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `internal/kernel/bus/bus.go`
- Create: `internal/kernel/bus/bus_test.go`

- [ ] **Step 1: 先写配置解析失败测试**

```go
func TestLoadRejectsMissingAccountSecrets(t *testing.T) {
    raw := []byte(`
system:
  log_level: info
accounts:
  - name: primary
    market_types: ["spot"]
`)

    _, err := LoadBytes(raw)
    require.Error(t, err)
    require.Contains(t, err.Error(), "api_key_env")
}
```

- [ ] **Step 2: 实现统一配置结构与校验**

```go
type Config struct {
    System     SystemConfig     `yaml:"system"`
    Accounts   []AccountConfig  `yaml:"accounts"`
    Strategies []StrategyConfig `yaml:"strategies"`
}

type AccountConfig struct {
    Name         string   `yaml:"name"`
    MarketTypes  []string `yaml:"market_types"`
    APIKeyEnv    string   `yaml:"api_key_env"`
    SecretKeyEnv string   `yaml:"secret_key_env"`
}

func (c Config) Validate() error {
    if len(c.Accounts) == 0 {
        return errors.New("at least one account is required")
    }
    for _, account := range c.Accounts {
        if account.APIKeyEnv == "" {
            return errors.New("api_key_env is required")
        }
        if account.SecretKeyEnv == "" {
            return errors.New("secret_key_env is required")
        }
    }
    return nil
}
```

- [ ] **Step 3: 定义核心领域模型**

```go
type MarketType string

const (
    MarketSpot      MarketType = "spot"
    MarketFuturesUM MarketType = "futures_um"
)

type PositionKey struct {
    Market       MarketType
    Symbol       string
    PositionSide string
}

type OrderIntent struct {
    StrategyID    string
    Market        MarketType
    Symbol        string
    PositionSide  string
    Side          string
    OrderType     string
    Price         string
    Quantity      string
    TimeInForce   string
    ReduceOnly    bool
    ClientOrderID string
    Reason        string
}
```

- [ ] **Step 4: 实现最小事件总线**

```go
type Event struct {
    Topic   string
    Payload any
}

type Bus struct {
    mu          sync.RWMutex
    subscribers map[string][]chan Event
}

func (b *Bus) Publish(event Event) {
    b.mu.RLock()
    defer b.mu.RUnlock()
    for _, ch := range b.subscribers[event.Topic] {
        select {
        case ch <- event:
        default:
        }
    }
}
```

- [ ] **Step 5: 跑领域和配置测试**

Run: `go test ./internal/config ./internal/domain ./internal/kernel/bus -v`  
Expected: PASS，验证配置校验、核心类型和事件总线行为。

- [ ] **Step 6: 提交统一模型层**

```bash
git add internal/config internal/domain internal/kernel/bus
git commit -m "feat: add config, domain model, and event bus"
```

### Task 3: 实现状态存储、限频器和风控基础

**Files:**
- Create: `internal/state/store.go`
- Create: `internal/state/store_test.go`
- Create: `internal/limit/limiter.go`
- Create: `internal/limit/limiter_test.go`
- Create: `internal/risk/service.go`
- Create: `internal/risk/service_test.go`

- [ ] **Step 1: 先写限频和风控失败用例**

```go
func TestLimiterRejectsWhenTokensExhausted(t *testing.T) {
    limiter := NewTokenBucket(1, time.Minute)
    require.NoError(t, limiter.Allow("orders"))
    require.ErrorIs(t, limiter.Allow("orders"), ErrRateLimited)
}

func TestRiskRejectsExposureAboveLimit(t *testing.T) {
    svc := Service{MaxStrategyNotional: decimal.RequireFromString("1000")}
    intent := domain.OrderIntent{StrategyID: "grid-btc", Symbol: "BTCUSDT", Quantity: "2", Price: "1000"}
    err := svc.Check(intent)
    require.ErrorContains(t, err, "strategy exposure")
}
```

- [ ] **Step 2: 实现线程安全状态仓**

```go
type Store struct {
    mu        sync.RWMutex
    orders    map[string]domain.OrderRecord
    positions map[domain.PositionKey]domain.PositionSnapshot
    balances  map[string]domain.Balance
}

func (s *Store) UpsertOrder(order domain.OrderRecord) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.orders[order.ClientOrderID] = order
}
```

- [ ] **Step 3: 实现本地令牌桶限频器**

```go
type TokenBucket struct {
    mu        sync.Mutex
    capacity  int
    tokens    int
    refillAt  time.Time
    interval  time.Duration
}

func (b *TokenBucket) Allow(_ string) error {
    b.mu.Lock()
    defer b.mu.Unlock()
    if time.Now().After(b.refillAt) {
        b.tokens = b.capacity
        b.refillAt = time.Now().Add(b.interval)
    }
    if b.tokens == 0 {
        return ErrRateLimited
    }
    b.tokens--
    return nil
}
```

- [ ] **Step 4: 实现硬风控服务**

```go
type Service struct {
    MaxStrategyNotional decimal.Decimal
}

func (s Service) Check(intent domain.OrderIntent) error {
    qty, err := decimal.NewFromString(intent.Quantity)
    if err != nil {
        return err
    }
    price, err := decimal.NewFromString(intent.Price)
    if err != nil {
        return err
    }
    if qty.Mul(price).GreaterThan(s.MaxStrategyNotional) {
        return fmt.Errorf("strategy exposure exceeds limit")
    }
    return nil
}
```

- [ ] **Step 5: 跑状态、限频和风控测试**

Run: `go test ./internal/state ./internal/limit ./internal/risk -v`  
Expected: PASS，覆盖并发安全、限频拒绝和风控拒绝行为。

- [ ] **Step 6: 提交状态与保护层基础能力**

```bash
git add internal/state internal/limit internal/risk
git commit -m "feat: add state store, limiter, and base risk checks"
```

### Task 4: 实现订单准入服务和 Binance 标准化适配层

**Files:**
- Create: `internal/admission/service.go`
- Create: `internal/admission/service_test.go`
- Create: `internal/exchange/binance/common/types.go`
- Create: `internal/exchange/binance/spot/adapter.go`
- Create: `internal/exchange/binance/spot/adapter_test.go`
- Create: `internal/exchange/binance/futures/adapter.go`
- Create: `internal/exchange/binance/futures/adapter_test.go`

- [ ] **Step 1: 先写准入流水线顺序测试**

```go
func TestAdmissionChecksRateLimitBeforeRisk(t *testing.T) {
    svc := NewService(fakeRules{}, fakeLimiter{err: limit.ErrRateLimited}, fakeRisk{})
    err := svc.Admit(domain.OrderIntent{ClientOrderID: "cid-1"})
    require.ErrorIs(t, err, limit.ErrRateLimited)
}
```

- [ ] **Step 2: 实现准入服务**

```go
type Service struct {
    rules   RuleValidator
    limiter Limiter
    risk    RiskChecker
}

func (s Service) Admit(intent domain.OrderIntent) error {
    if intent.ClientOrderID == "" {
        return errors.New("clientOrderID is required")
    }
    if err := s.rules.Validate(intent); err != nil {
        return err
    }
    if err := s.limiter.Allow("orders"); err != nil {
        return err
    }
    if err := s.risk.Check(intent); err != nil {
        return err
    }
    return nil
}
```

- [ ] **Step 3: 定义 Binance 公共报文结构**

```go
type WSOrderAck struct {
    ID     string `json:"id"`
    Status int    `json:"status"`
    Result struct {
        Symbol        string `json:"symbol"`
        ClientOrderID string `json:"clientOrderId"`
    } `json:"result"`
}
```

- [ ] **Step 4: 实现现货和合约适配器的标准化入口**

```go
type Adapter struct{}

func (a Adapter) NormalizeOrderAck(raw []byte) (domain.OrderEvent, error) {
    var ack common.WSOrderAck
    if err := json.Unmarshal(raw, &ack); err != nil {
        return domain.OrderEvent{}, err
    }
    return domain.OrderEvent{
        ClientOrderID: ack.Result.ClientOrderID,
        Symbol:        ack.Result.Symbol,
        Status:        "ACK",
    }, nil
}
```

- [ ] **Step 5: 跑准入与适配器测试**

Run: `go test ./internal/admission ./internal/exchange/binance/... -v`  
Expected: PASS，验证准入顺序和报文标准化正确。

- [ ] **Step 6: 提交准入和适配层**

```bash
git add internal/admission internal/exchange/binance
git commit -m "feat: add admission pipeline and binance adapters"
```

### Task 5: 实现网关服务、连接状态和恢复骨架

**Files:**
- Create: `internal/gateway/service.go`
- Create: `internal/gateway/service_test.go`

- [ ] **Step 1: 先写连接降级与恢复测试**

```go
func TestGatewayEntersDegradedModeOnUserStreamFailure(t *testing.T) {
    gw := NewService(nil, nil)
    gw.MarkUserStreamDown("primary")
    require.Equal(t, StateDegraded, gw.State("primary"))
}
```

- [ ] **Step 2: 实现网关状态机**

```go
type SessionState string

const (
    StateConnecting SessionState = "connecting"
    StateActive     SessionState = "active"
    StateDegraded   SessionState = "degraded"
    StateBackoff    SessionState = "backing_off"
    StateBanned     SessionState = "banned_until"
)
```

- [ ] **Step 3: 实现恢复入口和只减仓门控**

```go
type Service struct {
    mu         sync.RWMutex
    sessions   map[string]SessionState
    reduceOnly bool
}

func (s *Service) MarkUserStreamDown(account string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.sessions[account] = StateDegraded
    s.reduceOnly = true
}
```

- [ ] **Step 4: 暴露健康状态聚合接口**

```go
type Health struct {
    TradingEnabled bool
    ReduceOnly     bool
    Sessions       map[string]SessionState
}

func (s *Service) Health() Health {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return Health{TradingEnabled: !s.reduceOnly, ReduceOnly: s.reduceOnly, Sessions: maps.Clone(s.sessions)}
}
```

- [ ] **Step 5: 跑网关测试**

Run: `go test ./internal/gateway -v`  
Expected: PASS，验证用户流故障时进入 `degraded`，并开启 `reduce_only`。

- [ ] **Step 6: 提交网关骨架**

```bash
git add internal/gateway
git commit -m "feat: add gateway session state and recovery skeleton"
```

### Task 6: 实现策略运行时和网格策略状态机

**Files:**
- Create: `internal/strategy/runtime/runtime.go`
- Create: `internal/strategy/runtime/runtime_test.go`
- Create: `internal/strategy/grid/strategy.go`
- Create: `internal/strategy/grid/strategy_test.go`

- [ ] **Step 1: 先写策略运行时串行处理测试**

```go
func TestRuntimeProcessesStrategyEventsSequentially(t *testing.T) {
    rt := New()
    strategy := &fakeStrategy{}
    require.NoError(t, rt.Register(strategy))
    rt.DispatchMarketEvent(context.Background(), domain.MarketEvent{Symbol: "BTCUSDT"})
    rt.DispatchMarketEvent(context.Background(), domain.MarketEvent{Symbol: "BTCUSDT"})
    require.Equal(t, []string{"market", "market"}, strategy.calls)
}
```

- [ ] **Step 2: 实现运行时注册和串行派发**

```go
type Strategy interface {
    ID() string
    OnMarketEvent(context.Context, domain.MarketEvent) []domain.OrderIntent
    OnOrderEvent(context.Context, domain.OrderEvent) []domain.OrderIntent
    Snapshot() any
}

type Runtime struct {
    mu         sync.RWMutex
    strategies map[string]Strategy
}
```

- [ ] **Step 3: 先写网格状态迁移测试**

```go
func TestGridMovesFromBootstrappingToSeedingGrid(t *testing.T) {
    s := New(Config{Symbol: "BTCUSDT"})
    intents := s.OnMarketEvent(context.Background(), domain.MarketEvent{Symbol: "BTCUSDT", BestBid: "100"})
    require.Equal(t, StateSeedingGrid, s.state)
    require.NotEmpty(t, intents)
}
```

- [ ] **Step 4: 实现网格策略最小状态机**

```go
type State string

const (
    StateBootstrapping State = "Bootstrapping"
    StateSeedingGrid   State = "SeedingGrid"
    StateActive        State = "Active"
    StateReduceOnly    State = "ReduceOnly"
)

func (s *Strategy) OnMarketEvent(_ context.Context, event domain.MarketEvent) []domain.OrderIntent {
    if s.state == StateBootstrapping && event.Symbol == s.cfg.Symbol {
        s.state = StateSeedingGrid
        return s.seedOrders()
    }
    return nil
}
```

- [ ] **Step 5: 跑运行时和网格测试**

Run: `go test ./internal/strategy/runtime ./internal/strategy/grid -v`  
Expected: PASS，验证串行派发和网格初始布网状态切换。

- [ ] **Step 6: 提交策略层**

```bash
git add internal/strategy/runtime internal/strategy/grid
git commit -m "feat: add strategy runtime and grid plugin skeleton"
```

### Task 7: 组装主服务、健康接口与端到端集成测试

**Files:**
- Modify: `cmd/trader/main.go`
- Modify: `configs/example.yaml`
- Create: `internal/gateway/service_test.go`
- Create: `internal/strategy/runtime/runtime_test.go`

- [ ] **Step 1: 先写主服务装配测试**

```go
func TestBuildAppWiresCoreServices(t *testing.T) {
    app, err := BuildApp(testConfig())
    require.NoError(t, err)
    require.NotNil(t, app.gateway)
    require.NotNil(t, app.runtime)
}
```

- [ ] **Step 2: 在入口中装配配置、网关、运行时和健康接口**

```go
type app struct {
    gateway *gateway.Service
    runtime *runtime.Runtime
}

func BuildApp(cfg config.Config) (*app, error) {
    gw := gateway.NewService(nil, nil)
    rt := runtime.New()
    return &app{gateway: gw, runtime: rt}, nil
}
```

- [ ] **Step 3: 添加最小健康检查 HTTP 接口**

```go
http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
    _ = json.NewEncoder(w).Encode(app.gateway.Health())
})
```

- [ ] **Step 4: 扩展示例配置，补齐风控和策略参数**

```yaml
system:
  log_level: info
  http_addr: ":8080"
  reduce_only_on_degraded: true
strategies:
  - id: grid-btc
    type: grid
    market: futures_um
    symbol: BTCUSDT
    grid_count: 12
    upper_price: "120000"
    lower_price: "80000"
    base_order_size: "0.001"
```

- [ ] **Step 5: 跑全量验证**

Run: `gofmt -w cmd internal`  
Expected: 所有 Go 文件格式化完成。

Run: `go test ./...`  
Expected: PASS，所有单元与集成测试通过。

Run: `go run ./cmd/trader`  
Expected: 服务启动并暴露 `/healthz`。

- [ ] **Step 6: 提交第一版可运行骨架**

```bash
git add cmd/trader/main.go configs/example.yaml
git add internal
git commit -m "feat: wire trading service skeleton end-to-end"
```

## Self-Review Checklist

- spec 覆盖检查：
  本计划覆盖了配置、统一领域模型、WebSocket 优先网关骨架、限频、硬风控、订单准入、状态存储、策略运行时、网格插件、健康接口和测试策略。
- 占位扫描：
  不允许在实现过程中引入 `TODO`、`TBD`、`later`、`placeholder` 一类占位文本。
- 一致性检查：
  所有核心对象名称必须与 spec 保持一致，例如 `OrderIntent`、`State Store`、`Kill Switch`、`PositionKey`、`symbol + positionSide`。

## Execution Notes

- 当前工作区最初不是 Git 仓库，因此 Task 1 中先执行 `git init`。
- 若本地 Go 版本低于 `1.24`，先统一到兼容版本后再开始实现。
- 若 Binance WebSocket API 某些交易动作在目标环境受限，允许在实现中将相应发送器封装为接口，并暂时使用 REST fallback，但 fallback 只能挂在 gateway 层，不能泄漏到策略层。
