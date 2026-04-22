# Multi-Account Secret Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将多账号密钥管理从环境变量模式改为 `secret_ref + 本地 secret 文件目录` 模式，让账号列表由配置管理，真实 Binance 密钥由服务器受控目录管理。

**Architecture:** `config` 负责账号元数据；新增 `internal/secrets` 负责按 `secret_ref` 读取密钥；`cmd/trader` 启动流程按账号加载 secret 并启动连接；`gateway/debugview/web` 展示 `secretRef` 和 `secretStatus`，但不展示任何密钥值。

**Tech Stack:** Go 1.23+, standard library `encoding/json`, `os`, existing `github.com/stretchr/testify`.

---

## File Structure

本计划会创建或修改以下文件：

- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Create: `internal/secrets/provider.go`
- Create: `internal/secrets/file_provider.go`
- Create: `internal/secrets/file_provider_test.go`
- Modify: `internal/gateway/service.go`
- Modify: `internal/gateway/service_test.go`
- Modify: `internal/debugview/types.go`
- Modify: `internal/debugview/service.go`
- Modify: `internal/debugview/service_test.go`
- Modify: `cmd/trader/main.go`
- Modify: `cmd/trader/main_test.go`
- Modify: `configs/example.yaml`
- Modify: `.env.example`
- Modify: `deploy/README.md`
- Modify: `web/debug/app.js`

### Task 1: 调整配置结构为 secret_ref 模式

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `configs/example.yaml`

- [ ] **Step 1: 写失败测试，验证新配置结构**

```go
func TestLoadSupportsSecretRefAccounts(t *testing.T) {
    raw := []byte(`
system:
  log_level: info
  secret_dir: /etc/grid-trade/accounts
accounts:
  - name: primary
    enabled: true
    market_types: ["spot", "futures_um"]
    secret_ref: primary
`)

    cfg, err := LoadBytes(raw)

    require.NoError(t, err)
    require.Equal(t, "/etc/grid-trade/accounts", cfg.System.SecretDir)
    require.True(t, cfg.Accounts[0].Enabled)
    require.Equal(t, "primary", cfg.Accounts[0].SecretRef)
}
```

- [ ] **Step 2: 写失败测试，验证缺少 secret_ref 会报错**

```go
func TestLoadRejectsMissingSecretRef(t *testing.T) {
    raw := []byte(`
accounts:
  - name: primary
    enabled: true
    market_types: ["spot"]
`)

    _, err := LoadBytes(raw)

    require.ErrorContains(t, err, "secret_ref")
}
```

- [ ] **Step 3: 运行配置测试确认失败**

Run: `go test ./internal/config -v`  
Expected: FAIL，提示 `SecretDir`、`Enabled` 或 `SecretRef` 未定义。

- [ ] **Step 4: 修改配置类型**

```go
type SystemConfig struct {
    LogLevel  string `yaml:"log_level"`
    HTTPAddr  string `yaml:"http_addr"`
    SecretDir string `yaml:"secret_dir"`
}

type AccountConfig struct {
    Name        string   `yaml:"name"`
    Enabled     bool     `yaml:"enabled"`
    MarketTypes []string `yaml:"market_types"`
    SecretRef   string   `yaml:"secret_ref"`
}
```

- [ ] **Step 5: 修改配置校验**

```go
func (c Config) Validate() error {
    if len(c.Accounts) == 0 {
        return errors.New("at least one account is required")
    }
    names := map[string]struct{}{}
    for _, account := range c.Accounts {
        if account.Name == "" {
            return errors.New("account name is required")
        }
        if _, exists := names[account.Name]; exists {
            return fmt.Errorf("duplicate account name %q", account.Name)
        }
        names[account.Name] = struct{}{}
        if !account.Enabled {
            continue
        }
        if account.SecretRef == "" {
            return errors.New("secret_ref is required")
        }
        if len(account.MarketTypes) == 0 {
            return errors.New("market_types is required")
        }
    }
    return nil
}
```

- [ ] **Step 6: 更新 `configs/example.yaml`**

```yaml
system:
  log_level: info
  http_addr: ":8080"
  secret_dir: "/etc/grid-trade/accounts"
accounts:
  - name: primary
    enabled: true
    market_types: ["spot", "futures_um"]
    secret_ref: primary
```

- [ ] **Step 7: 跑配置测试**

Run: `go test ./internal/config -v`  
Expected: PASS。

- [ ] **Step 8: 提交配置结构改造**

```bash
git add internal/config configs/example.yaml
git commit -m "feat: switch account config to secret refs"
```

### Task 2: 实现本地文件 SecretProvider

**Files:**
- Create: `internal/secrets/provider.go`
- Create: `internal/secrets/file_provider.go`
- Create: `internal/secrets/file_provider_test.go`

- [ ] **Step 1: 写失败测试，验证能读取 secret JSON**

```go
func TestFileProviderLoadsAccountSecret(t *testing.T) {
    dir := t.TempDir()
    require.NoError(t, os.WriteFile(filepath.Join(dir, "primary.json"), []byte(`{
      "api_key": "api",
      "secret_key": "secret"
    }`), 0o600))

    provider := NewFileProvider(dir)

    secret, err := provider.Load(context.Background(), "primary")

    require.NoError(t, err)
    require.Equal(t, "api", secret.APIKey)
    require.Equal(t, "secret", secret.SecretKey)
}
```

- [ ] **Step 2: 写失败测试，验证缺字段会报错且不泄露值**

```go
func TestFileProviderRejectsMissingSecretFields(t *testing.T) {
    dir := t.TempDir()
    require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{"api_key":"api"}`), 0o600))

    provider := NewFileProvider(dir)

    _, err := provider.Load(context.Background(), "bad")

    require.ErrorContains(t, err, "secret_key")
    require.NotContains(t, err.Error(), "api")
}
```

- [ ] **Step 3: 运行 secrets 测试确认失败**

Run: `go test ./internal/secrets -v`  
Expected: FAIL，package 或 provider 未定义。

- [ ] **Step 4: 定义抽象类型**

```go
type AccountSecret struct {
    APIKey    string
    SecretKey string
}

type Provider interface {
    Load(ctx context.Context, ref string) (AccountSecret, error)
}
```

- [ ] **Step 5: 实现 `FileProvider`**

```go
type FileProvider struct {
    baseDir string
}

func NewFileProvider(baseDir string) FileProvider {
    return FileProvider{baseDir: baseDir}
}
```

- [ ] **Step 6: 实现 JSON 读取和字段校验**

```go
func (p FileProvider) Load(ctx context.Context, ref string) (AccountSecret, error) {
    path := filepath.Join(p.baseDir, ref+".json")
    raw, err := os.ReadFile(path)
    if err != nil {
        return AccountSecret{}, fmt.Errorf("secret load failed: %w", err)
    }
    var payload struct {
        APIKey string `json:"api_key"`
        SecretKey string `json:"secret_key"`
    }
    if err := json.Unmarshal(raw, &payload); err != nil {
        return AccountSecret{}, fmt.Errorf("secret load failed: invalid json")
    }
    if payload.APIKey == "" {
        return AccountSecret{}, errors.New("secret load failed: api_key is required")
    }
    if payload.SecretKey == "" {
        return AccountSecret{}, errors.New("secret load failed: secret_key is required")
    }
    return AccountSecret{APIKey: payload.APIKey, SecretKey: payload.SecretKey}, nil
}
```

- [ ] **Step 7: 跑 secrets 测试**

Run: `go test ./internal/secrets -v`  
Expected: PASS。

- [ ] **Step 8: 提交 secret provider**

```bash
git add internal/secrets
git commit -m "feat: add file secret provider"
```

### Task 3: 扩展 Gateway 和 DebugView 的 secret 状态

**Files:**
- Modify: `internal/gateway/service.go`
- Modify: `internal/gateway/service_test.go`
- Modify: `internal/debugview/types.go`
- Modify: `internal/debugview/service.go`
- Modify: `internal/debugview/service_test.go`

- [ ] **Step 1: 写失败测试，验证 gateway 快照包含 secret 状态**

```go
func TestGatewaySnapshotIncludesSecretState(t *testing.T) {
    gw := NewService(nil, nil)
    gw.MarkSecretState("primary", domain.MarketSpot, "primary", "load_failed", "secret load failed")

    snapshot := gw.Snapshot()

    require.Equal(t, "primary", snapshot.Accounts[0].SecretRef)
    require.Equal(t, "load_failed", snapshot.Accounts[0].SecretStatus)
}
```

- [ ] **Step 2: 实现 gateway secret 状态字段**

```go
SecretRef string
SecretStatus string
SecretError string
```

Add maps:

```go
secretRefs map[accountMarketKey]string
secretStatuses map[accountMarketKey]string
secretErrors map[accountMarketKey]string
```

- [ ] **Step 3: 实现 `MarkSecretState`**

```go
func (s *Service) MarkSecretState(account string, market domain.MarketType, ref, status, errSummary string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    key := accountMarketKey{account: account, market: market}
    s.secretRefs[key] = ref
    s.secretStatuses[key] = status
    s.secretErrors[key] = errSummary
    if status == "load_failed" {
        s.markets[key] = StateDegraded
        s.lastErrors[key] = errSummary
        s.reduceOnly = true
    }
}
```

- [ ] **Step 4: 写失败测试，验证 debugview 生成密钥加载失败告警**

```go
func TestDashboardIncludesSecretLoadFailedAlert(t *testing.T) {
    svc := NewService(fakeGatewaySnapshot{accounts: []gateway.AccountSnapshot{
        {Account: "primary", Market: domain.MarketSpot, SecretRef: "primary", SecretStatus: "load_failed", LastError: "secret load failed"},
    }})

    dashboard := svc.Dashboard()

    require.Equal(t, "密钥加载失败", dashboard.Alerts[0].Title)
}
```

- [ ] **Step 5: 扩展 debugview account row**

```go
SecretRef string `json:"secretRef"`
SecretStatus string `json:"secretStatus"`
```

- [ ] **Step 6: 跑 gateway/debugview 测试**

Run: `go test ./internal/gateway ./internal/debugview -v`  
Expected: PASS。

- [ ] **Step 7: 提交 secret 状态展示模型**

```bash
git add internal/gateway internal/debugview
git commit -m "feat: expose account secret status in debug view"
```

### Task 4: 启动流程改为按 secret_ref 加载密钥

**Files:**
- Modify: `cmd/trader/main.go`
- Modify: `cmd/trader/main_test.go`

- [ ] **Step 1: 写失败测试，验证缺失 secret 文件时账号 degraded**

```go
func TestBuildAppMarksAccountDegradedWhenSecretFileMissing(t *testing.T) {
    cfg := config.Config{
        System: config.SystemConfig{SecretDir: t.TempDir()},
        Accounts: []config.AccountConfig{{Name: "primary", Enabled: true, MarketTypes: []string{"spot"}, SecretRef: "primary"}},
    }

    app, err := BuildApp(cfg)
    require.NoError(t, err)

    snapshot := app.gateway.Snapshot()
    require.Equal(t, gateway.StateDegraded, snapshot.Accounts[0].SessionState)
    require.Equal(t, "primary", snapshot.Accounts[0].SecretRef)
    require.Equal(t, "load_failed", snapshot.Accounts[0].SecretStatus)
}
```

- [ ] **Step 2: 写测试，验证 disabled 账号不启动**

```go
func TestBuildAppSkipsDisabledAccounts(t *testing.T) {
    cfg := config.Config{Accounts: []config.AccountConfig{{Name: "disabled", Enabled: false, MarketTypes: []string{"spot"}, SecretRef: "disabled"}}}
    app, err := BuildApp(cfg)
    require.NoError(t, err)
    require.Empty(t, app.gateway.Snapshot().Accounts)
}
```

- [ ] **Step 3: 修改 `startConfiguredAccounts`**

```go
provider := secrets.NewFileProvider(cfg.System.SecretDir)
secret, err := provider.Load(context.Background(), account.SecretRef)
```

失败时：

```go
gw.MarkSecretState(account.Name, market, account.SecretRef, "load_failed", err.Error())
```

成功时：

```go
gw.MarkSecretState(account.Name, market, account.SecretRef, "loaded", "")
```

- [ ] **Step 4: 删除环境变量密钥读取逻辑**

Remove:

```go
apiKey := os.Getenv(account.APIKeyEnv)
secretKey := os.Getenv(account.SecretKeyEnv)
```

- [ ] **Step 5: 跑 cmd 测试**

Run: `go test ./cmd/trader -v`  
Expected: PASS。

- [ ] **Step 6: 提交启动流程改造**

```bash
git add cmd/trader
git commit -m "feat: load account secrets from secret provider"
```

### Task 5: 更新 Debug Console 页面和部署文档

**Files:**
- Modify: `web/debug/app.js`
- Modify: `configs/example.yaml`
- Modify: `.env.example`
- Modify: `deploy/README.md`

- [ ] **Step 1: 更新页面账户行展示**

Add:

```javascript
<div class="account-meta">密钥引用：${escapeHTML(account.secretRef || "-")}</div>
<div class="account-meta">密钥状态：${escapeHTML(labelSecretStatus(account.secretStatus))}</div>
```

- [ ] **Step 2: 增加密钥状态中文映射**

```javascript
function labelSecretStatus(value) {
  if (value === "loaded") return "已加载";
  if (value === "load_failed") return "加载失败";
  if (value === "not_configured") return "未配置";
  return value || "-";
}
```

- [ ] **Step 3: 更新 `configs/example.yaml`**

```yaml
system:
  log_level: info
  http_addr: ":8080"
  secret_dir: "/etc/grid-trade/accounts"
accounts:
  - name: primary
    enabled: true
    market_types: ["spot", "futures_um"]
    secret_ref: primary
```

- [ ] **Step 4: 更新 `.env.example`**

Remove Binance key placeholders. Keep:

```bash
TRADER_CONFIG=/opt/grid-trade/configs/example.yaml
TRADER_HTTP_ADDR=:18080
```

- [ ] **Step 5: 更新 `deploy/README.md` 增加 secret JSON 示例**

```json
{
  "api_key": "your-real-key",
  "secret_key": "your-real-secret"
}
```

- [ ] **Step 6: 检查页面不泄露 secret 字段**

Run: `rg -n "api_key|secret_key|your-real-key|your-real-secret" web/debug internal/debugview internal/debughttp`  
Expected: 无输出。

- [ ] **Step 7: 提交页面和部署文档**

```bash
git add web/debug configs/example.yaml .env.example deploy/README.md
git commit -m "docs: update deployment for account secret files"
```

### Task 6: 验证、重启和推送

**Files:**
- No new files

- [ ] **Step 1: 全量格式化**

Run: `gofmt -w cmd internal`  
Expected: 无输出。

- [ ] **Step 2: 全量测试**

Run: `go test ./...`  
Expected: PASS。

- [ ] **Step 3: 本地运行验证缺失 secret 文件状态**

Run: `TRADER_HTTP_ADDR=:18080 go run ./cmd/trader`  
Expected: 服务启动。

Run: `curl -s http://127.0.0.1:18080/debug/dashboard`  
Expected: 返回含有 `密钥加载失败` 告警。

- [ ] **Step 4: 推送分支**

```bash
git push
```

Expected: `feat/ws-trading-skeleton` 推送成功。

## Self-Review Checklist

- spec 覆盖检查：
  本计划覆盖配置结构、secret provider、启动流程、gateway/debugview 状态、页面展示和部署文档。
- 安全检查：
  页面、debug API、日志和配置中不得展示真实密钥。
- 占位扫描：
  计划和实现中不能引入未完成占位，例如 `TODO`、`TBD`、`placeholder`、`待定`。
- 兼容性检查：
  多账号中某个账号 secret 加载失败时，不能影响其他账号继续启动。
