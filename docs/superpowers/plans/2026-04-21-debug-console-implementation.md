# Debug Console Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在现有 Go 交易服务中内置一个只读调试控制台，用于查看 Spot/Futures 连接健康、混合告警、账户用户流状态和交易保护状态。

**Architecture:** 后端新增 `debugview` 展示模型聚合层和 `debughttp` HTTP handler；`gateway` 暴露只读快照，避免前端绑定内部结构；前端使用内置静态 HTML/CSS/JS，通过 `/debug/dashboard` 和 `/debug/accounts` 渲染运维看板。

**Tech Stack:** Go 1.23+, standard library `net/http` and `embed`, existing `github.com/stretchr/testify`, plain HTML/CSS/JavaScript.

---

## File Structure

本计划会创建或修改以下文件：

- Modify: `internal/gateway/service.go`
- Modify: `internal/gateway/service_test.go`
- Create: `internal/debugview/types.go`
- Create: `internal/debugview/service.go`
- Create: `internal/debugview/service_test.go`
- Create: `internal/debughttp/handler.go`
- Create: `internal/debughttp/handler_test.go`
- Create: `web/debug/index.html`
- Create: `web/debug/app.css`
- Create: `web/debug/app.js`
- Modify: `cmd/trader/main.go`
- Modify: `cmd/trader/main_test.go`

### Task 1: 给 Gateway 增加只读状态快照

**Files:**
- Modify: `internal/gateway/service.go`
- Modify: `internal/gateway/service_test.go`

- [ ] **Step 1: 写失败测试，验证 gateway 能导出账户市场快照**

```go
func TestGatewaySnapshotIncludesAccountMarketState(t *testing.T) {
    gw := NewService(fakeListenKeyProvider{}, fakeConnector{})
    err := gw.BootstrapAccount(context.Background(), "primary", []domain.MarketType{domain.MarketSpot})
    require.NoError(t, err)

    snapshot := gw.Snapshot()

    require.Len(t, snapshot.Accounts, 1)
    require.Equal(t, "primary", snapshot.Accounts[0].Account)
    require.Equal(t, domain.MarketSpot, snapshot.Accounts[0].Market)
    require.Equal(t, StateActive, snapshot.Accounts[0].SessionState)
    require.False(t, snapshot.Accounts[0].ReduceOnly)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/gateway -run TestGatewaySnapshotIncludesAccountMarketState -v`  
Expected: FAIL，提示 `Snapshot` 或 `GatewaySnapshot` 未定义。

- [ ] **Step 3: 实现 Gateway 只读快照类型**

```go
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
```

- [ ] **Step 4: 在 `BootstrapAccount` 中记录每个 account/market 的状态**

```go
type accountMarketKey struct {
    account string
    market  domain.MarketType
}

marketSessions map[accountMarketKey]SessionState
```

- [ ] **Step 5: 实现 `Snapshot()`，返回拷贝而不是内部 map**

```go
func (s *Service) Snapshot() Snapshot {
    s.mu.RLock()
    defer s.mu.RUnlock()

    accounts := make([]AccountSnapshot, 0, len(s.marketSessions))
    for key, state := range s.marketSessions {
        accounts = append(accounts, AccountSnapshot{
            Account:      key.account,
            Market:       key.market,
            SessionState: state,
            ReduceOnly:   s.reduceOnly,
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
```

- [ ] **Step 6: 运行 gateway 测试**

Run: `go test ./internal/gateway -v`  
Expected: PASS。

- [ ] **Step 7: 提交 gateway 快照能力**

```bash
git add internal/gateway/service.go internal/gateway/service_test.go
git commit -m "feat: expose gateway debug snapshot"
```

### Task 2: 实现 debugview 展示模型和聚合服务

**Files:**
- Create: `internal/debugview/types.go`
- Create: `internal/debugview/service.go`
- Create: `internal/debugview/service_test.go`

- [ ] **Step 1: 写失败测试，验证 degraded futures 生成告警和市场卡片**

```go
func TestServiceBuildsDashboardFromGatewaySnapshot(t *testing.T) {
    svc := NewService(fakeGatewaySnapshot{
        accounts: []gateway.AccountSnapshot{
            {Account: "primary", Market: domain.MarketFuturesUM, SessionState: gateway.StateDegraded, ReduceOnly: true, LastError: "listen key stale"},
            {Account: "primary", Market: domain.MarketSpot, SessionState: gateway.StateActive},
        },
    })

    dashboard := svc.Dashboard()

    require.Len(t, dashboard.Alerts, 2)
    require.Equal(t, domain.MarketSpot, dashboard.Markets[0].Market)
    require.Equal(t, domain.MarketFuturesUM, dashboard.Markets[1].Market)
    require.Equal(t, HealthDegraded, dashboard.Markets[1].Health)
    require.Equal(t, 1, dashboard.Markets[1].ReduceOnlyAccounts)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/debugview -v`  
Expected: FAIL，提示 package 或类型未定义。

- [ ] **Step 3: 定义展示模型类型**

```go
type Severity string
type AlertCategory string
type Health string
type AlertStatus string

const (
    SeverityInfo     Severity = "info"
    SeverityWarning  Severity = "warning"
    SeverityCritical Severity = "critical"
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
```

- [ ] **Step 4: 定义 dashboard 和 market card 类型**

```go
type Dashboard struct {
    Alerts  []AlertSummary     `json:"alerts"`
    Markets []MarketHealthCard `json:"markets"`
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
```

- [ ] **Step 5: 实现 `Service.Dashboard()` 聚合逻辑**

```go
type GatewaySnapshotter interface {
    Snapshot() gateway.Snapshot
}

type Service struct {
    gateway GatewaySnapshotter
    now     func() time.Time
}

func (s Service) Dashboard() Dashboard {
    snapshot := s.gateway.Snapshot()
    markets := buildMarketCards(snapshot.Accounts)
    alerts := buildAlerts(snapshot.Accounts, s.now())
    return Dashboard{Alerts: alerts, Markets: markets}
}
```

- [ ] **Step 6: 增加账户明细排序测试**

```go
func TestAccountsSortsProblemRowsFirst(t *testing.T) {
    svc := NewService(fakeGatewaySnapshot{accounts: []gateway.AccountSnapshot{
        {Account: "ok", Market: domain.MarketSpot, SessionState: gateway.StateActive},
        {Account: "bad", Market: domain.MarketSpot, SessionState: gateway.StateDegraded},
    }})

    rows := svc.Accounts(domain.MarketSpot)

    require.Equal(t, "bad", rows[0].Account)
    require.Equal(t, "ok", rows[1].Account)
}
```

- [ ] **Step 7: 实现 `AccountConnectionRow` 和 `Accounts(market)`**

```go
type AccountConnectionRow struct {
    Account          string               `json:"account"`
    SessionState     gateway.SessionState `json:"sessionState"`
    UserStreamState  string               `json:"userStreamState"`
    ListenKeyState   string               `json:"listenKeyState"`
    ListenKeyExpiresAt *time.Time         `json:"listenKeyExpiresAt,omitempty"`
    LastHeartbeatAt  *time.Time           `json:"lastHeartbeatAt,omitempty"`
    LastReconnectAt  *time.Time           `json:"lastReconnectAt,omitempty"`
    ReduceOnly       bool                 `json:"reduceOnly"`
    KillSwitch       bool                 `json:"killSwitch"`
    ActiveBackoff    bool                 `json:"activeBackoff"`
    LastError        string               `json:"lastError"`
}
```

- [ ] **Step 8: 运行 debugview 测试**

Run: `go test ./internal/debugview -v`  
Expected: PASS。

- [ ] **Step 9: 提交 debugview 聚合层**

```bash
git add internal/debugview
git commit -m "feat: add debug dashboard view model"
```

### Task 3: 实现 debug HTTP API

**Files:**
- Create: `internal/debughttp/handler.go`
- Create: `internal/debughttp/handler_test.go`

- [ ] **Step 1: 写失败测试，验证 `/debug/dashboard` 返回 JSON**

```go
func TestDashboardEndpointReturnsJSON(t *testing.T) {
    handler := NewHandler(fakeView{
        dashboard: debugview.Dashboard{Markets: []debugview.MarketHealthCard{{Market: domain.MarketSpot}}},
    })

    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/debug/dashboard", nil)
    handler.ServeHTTP(rr, req)

    require.Equal(t, http.StatusOK, rr.Code)
    require.Contains(t, rr.Header().Get("Content-Type"), "application/json")
    require.Contains(t, rr.Body.String(), `"markets"`)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/debughttp -v`  
Expected: FAIL，提示 package 或 handler 未定义。

- [ ] **Step 3: 定义 debug HTTP view 接口和 handler**

```go
type View interface {
    Dashboard() debugview.Dashboard
    Accounts(domain.MarketType) []debugview.AccountConnectionRow
    Events(domain.MarketType, string) []debugview.Event
}

type Handler struct {
    view View
    mux  *http.ServeMux
}
```

- [ ] **Step 4: 实现 `/debug/dashboard`**

```go
func (h *Handler) dashboard(w http.ResponseWriter, _ *http.Request) {
    writeJSON(w, http.StatusOK, h.view.Dashboard())
}
```

- [ ] **Step 5: 写并实现 `/debug/accounts?market=spot` 测试**

```go
func TestAccountsEndpointRequiresMarket(t *testing.T) {
    handler := NewHandler(fakeView{})
    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/debug/accounts", nil)

    handler.ServeHTTP(rr, req)

    require.Equal(t, http.StatusBadRequest, rr.Code)
}
```

- [ ] **Step 6: 实现 `/debug/accounts` 参数校验和响应**

```go
func (h *Handler) accounts(w http.ResponseWriter, r *http.Request) {
    market := domain.MarketType(r.URL.Query().Get("market"))
    if market == "" {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "market is required"})
        return
    }
    writeJSON(w, http.StatusOK, AccountsResponse{Market: market, Accounts: h.view.Accounts(market)})
}
```

- [ ] **Step 7: 实现 `/debug/events` 空结果接口**

```go
type EventsResponse struct {
    Events []debugview.Event `json:"events"`
}
```

- [ ] **Step 8: 运行 debughttp 测试**

Run: `go test ./internal/debughttp -v`  
Expected: PASS。

- [ ] **Step 9: 提交 debug HTTP API**

```bash
git add internal/debughttp
git commit -m "feat: add debug console http api"
```

### Task 4: 实现内置静态调试页面

**Files:**
- Create: `web/debug/index.html`
- Create: `web/debug/app.css`
- Create: `web/debug/app.js`

- [ ] **Step 1: 创建只读调试台 HTML**

```html
<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Grid Trade Debug Console</title>
  <link rel="stylesheet" href="/debug/static/app.css">
</head>
<body>
  <main class="shell">
    <header class="hero">
      <p class="eyebrow">Binance WS Trading Service</p>
      <h1>连接运维看板</h1>
      <p id="refresh-state">等待数据...</p>
    </header>
    <section id="alerts" class="alerts"></section>
    <section id="markets" class="markets"></section>
  </main>
  <script src="/debug/static/app.js"></script>
</body>
</html>
```

- [ ] **Step 2: 创建页面样式**

```css
:root {
  --bg: #f3f0e8;
  --panel: #fffaf0;
  --ink: #1c1917;
  --muted: #78716c;
  --green: #067647;
  --amber: #b54708;
  --red: #b42318;
  --line: #ded7c8;
}

body {
  margin: 0;
  font-family: ui-serif, Georgia, "Times New Roman", serif;
  background: radial-gradient(circle at top left, #fff7d6, transparent 34%), var(--bg);
  color: var(--ink);
}
```

- [ ] **Step 3: 创建 JS 轮询和渲染逻辑**

```javascript
async function loadDashboard() {
  const response = await fetch('/debug/dashboard');
  if (!response.ok) throw new Error(`dashboard ${response.status}`);
  return response.json();
}

function renderAlerts(alerts) {
  const root = document.getElementById('alerts');
  if (!alerts || alerts.length === 0) {
    root.innerHTML = '<div class="empty">No active alerts</div>';
    return;
  }
  root.innerHTML = alerts.map(alert => `
    <article class="alert ${alert.severity}">
      <strong>${escapeHTML(alert.title)}</strong>
      <span>${escapeHTML(alert.market)} / ${escapeHTML(alert.account || 'system')}</span>
      <p>${escapeHTML(alert.detail || '')}</p>
    </article>
  `).join('');
}
```

- [ ] **Step 4: 确认页面中没有操作按钮**

Run: `rg -n "button|下单|撤单|恢复|kill switch off|resume" web/debug`  
Expected: 不出现交易操作按钮；如果出现 `button`，必须确认只用于展开只读内容。

- [ ] **Step 5: 提交静态页面**

```bash
git add web/debug
git commit -m "feat: add debug console static page"
```

### Task 5: 挂载 debug API 和静态页面到主服务

**Files:**
- Modify: `cmd/trader/main.go`
- Modify: `cmd/trader/main_test.go`

- [ ] **Step 1: 写失败测试，验证 app 挂载 debug handler**

```go
func TestBuildAppMountsDebugDashboard(t *testing.T) {
    app, err := BuildApp(config.Config{})
    require.NoError(t, err)

    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/debug/dashboard", nil)
    app.handler.ServeHTTP(rr, req)

    require.Equal(t, http.StatusOK, rr.Code)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./cmd/trader -run TestBuildAppMountsDebugDashboard -v`  
Expected: FAIL，提示 `handler` 未定义或路径未挂载。

- [ ] **Step 3: 在 app 结构中加入 `handler *http.ServeMux`**

```go
type app struct {
    gateway *gateway.Service
    runtime *runtimepkg.Runtime
    httpAddr string
    handler *http.ServeMux
}
```

- [ ] **Step 4: 在 `BuildApp` 中装配 debugview 和 debughttp**

```go
view := debugview.NewService(gw)
debugHandler := debughttp.NewHandler(view)
mux := http.NewServeMux()
mux.Handle("/debug/", debugHandler)
mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
    write health JSON
})
```

- [ ] **Step 5: 修改 `main` 使用 `app.handler`**

```go
server := &http.Server{Addr: app.httpAddr, Handler: app.handler}
go func() {
    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Printf("http server stopped: %v", err)
    }
}()
```

- [ ] **Step 6: 运行 cmd 测试**

Run: `go test ./cmd/trader -v`  
Expected: PASS。

- [ ] **Step 7: 提交主服务挂载**

```bash
git add cmd/trader/main.go cmd/trader/main_test.go
git commit -m "feat: mount debug console in trader service"
```

### Task 6: 端到端验证和推送

**Files:**
- No new files

- [ ] **Step 1: 全量格式化**

Run: `gofmt -w cmd internal`  
Expected: 无输出，Go 文件格式化完成。

- [ ] **Step 2: 全量测试**

Run: `go test ./...`  
Expected: PASS。

- [ ] **Step 3: 启动服务并验证 debug API**

Run: `TRADER_HTTP_ADDR=:18080 go run ./cmd/trader`  
Expected: 服务持续运行。

Run in another shell: `curl -s http://127.0.0.1:18080/debug/dashboard`  
Expected: 返回 JSON，包含 `alerts` 和 `markets`。

Run in another shell: `curl -s http://127.0.0.1:18080/debug/`  
Expected: 返回 HTML，包含 `连接运维看板`。

- [ ] **Step 4: 检查页面不包含交易操作按钮**

Run: `rg -n "下单|撤单|恢复交易|关闭风控|开仓|平仓" web/debug`  
Expected: 无输出。

- [ ] **Step 5: 推送分支**

```bash
git push
```

Expected: `feat/ws-trading-skeleton` 推送成功。

## Self-Review Checklist

- spec 覆盖检查：
  本计划覆盖了 debugview 展示模型、gateway 快照、debug HTTP API、内置静态页面、主服务挂载和端到端验证。
- 占位扫描：
  计划和实现中不能引入 `TODO`、`TBD`、`placeholder`、`待定`。
- 安全边界：
  第一版页面只读，不能出现交易操作按钮或恢复交易入口。
- 一致性检查：
  字段命名必须和 spec 一致，例如 `AlertSummary`、`MarketHealthCard`、`AccountConnectionRow`、`reduceOnly`、`listenKeyState`。
