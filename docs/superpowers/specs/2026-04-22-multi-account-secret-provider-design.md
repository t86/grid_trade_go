# 多账号 Secret Provider 设计

## 概述

本文档定义多账号场景下的账号配置与密钥加载设计。目标是将“账号列表管理”和“密钥值管理”解耦，使 Binance 交易服务能够在服务器环境下支持多账号，并避免把真实密钥写入仓库、配置文件或调试页面。

第一版采用本地文件目录形式的 secret provider。账号元数据放在业务配置中，真实密钥放在服务器受控目录中，由服务启动时按账号动态读取。

## 目标

- 支持多账号配置，而不是只围绕一组环境变量工作。
- 将账号元数据和密钥值解耦。
- 让账号增删改不依赖修改服务代码。
- 保证真实密钥不进入仓库、不进入调试台、不进入日志。
- 支持启动时按账号加载密钥，并将失败状态反馈到 debug console。
- 为后续接入云 Secret Manager 保留统一抽象。

## 非目标

- 第一版不支持网页端增删账号。
- 第一版不支持热更新账号列表。
- 第一版不实现云 Secret Manager。
- 第一版不实现自动轮换密钥。
- 第一版不在数据库中管理账号配置。

## 设计原则

- `账号是谁` 由业务配置决定。
- `密钥是什么` 由 secret provider 决定。
- 仓库中不保存任何真实 Binance 密钥。
- 服务启动失败不应由单个账号密钥问题拖垮全部账号。
- 调试控制台最多展示密钥引用和加载状态，不展示密钥内容。
- secret provider 抽象必须可替换，便于后续接入其他后端。

## 配置模型

账号配置应从环境变量名模式切换为 secret reference 模式。

旧结构：

```yaml
accounts:
  - name: primary
    market_types: ["spot", "futures_um"]
    api_key_env: BINANCE_API_KEY
    secret_key_env: BINANCE_SECRET_KEY
```

新结构：

```yaml
system:
  http_addr: ":18080"
  secret_dir: "/etc/grid-trade/accounts"

accounts:
  - name: primary
    enabled: true
    market_types: ["spot", "futures_um"]
    secret_ref: primary

  - name: maker_01
    enabled: true
    market_types: ["spot"]
    secret_ref: maker_01
```

字段定义：

- `system.secret_dir`
  服务器 secret 目录。
- `accounts[].name`
  账号唯一标识。
- `accounts[].enabled`
  是否启用该账号。
- `accounts[].market_types`
  启用的市场类型。
- `accounts[].secret_ref`
  密钥引用名，不是文件路径，不是环境变量名。

约束：

- `name` 必须唯一。
- `enabled: false` 的账号不会参与启动。
- `secret_ref` 不能为空。
- `market_types` 只能使用已支持的市场值。

## Secret Provider 抽象

引入统一接口：

```go
type SecretProvider interface {
    Load(ctx context.Context, ref string) (AccountSecret, error)
}
```

配套数据结构：

```go
type AccountSecret struct {
    APIKey    string
    SecretKey string
}
```

抽象职责：

- 根据 `ref` 加载某个账号的真实密钥。
- 不关心账号启用了哪些市场。
- 不参与交易逻辑。
- 不写日志泄露密钥。

## 本地文件实现

第一版实现 `FileSecretProvider`：

```go
type FileSecretProvider struct {
    BaseDir string
}
```

加载规则：

- `ref = primary` 时，读取 `${BaseDir}/primary.json`
- `ref = maker_01` 时，读取 `${BaseDir}/maker_01.json`

推荐目录：

- `/etc/grid-trade/accounts`

推荐文件名：

- `/etc/grid-trade/accounts/primary.json`
- `/etc/grid-trade/accounts/maker_01.json`

推荐文件内容：

```json
{
  "api_key": "real-api-key",
  "secret_key": "real-secret-key"
}
```

文件权限要求：

- 权限建议 `600`
- 所属用户应为服务运行用户
- 目录权限应限制为管理员和服务用户可访问

错误处理：

- 文件不存在：返回明确错误
- JSON 无法解析：返回明确错误
- `api_key` 或 `secret_key` 为空：返回明确错误

## 启动流程

启动流程调整为：

1. 读取主配置。
2. 初始化 `SecretProvider`。
3. 遍历所有 `enabled` 账号。
4. 对每个账号：
   - 根据 `secret_ref` 加载密钥
   - 成功则按 `market_types` 启动用户流和后续交易连接
   - 失败则将该账号各市场标记为 `degraded`
5. 其他账号继续启动，不受单账号失败影响。

关键要求：

- 不能因为一个账号 secret 文件损坏而让整个服务退出。
- 启动时必须对每个失败账号留下可观测状态。

## Gateway 与运行态状态变化

为了支持调试控制台，gateway 或启动协调层需要新增以下状态：

- `secretRef`
- `secretStatus`
- `secretError`

其中：

- `secretStatus`
  可取 `loaded`、`load_failed`、`not_configured`
- `secretError`
  保存错误摘要，不包含敏感值

当 secret 加载失败时：

- `sessionState` 应标记为 `degraded`
- `reduceOnly` 可开启
- `lastError` 应更新为 secret 加载失败摘要

## Debug Console 展示变化

调试控制台应补充以下展示：

### 账户明细新增字段

- `secretRef`
- `secretStatus`

页面示例：

- `密钥引用：primary`
- `密钥状态：已加载`
- `密钥状态：加载失败`

### 新增告警类型

新增一类连接前置依赖告警：

- `secret load failed`

页面上显示为：

- 标题：`密钥加载失败`
- 详情：例如 `secret load failed: file not found`

禁止展示：

- 完整文件路径
- API key 明文
- Secret key 明文
- 密钥字段片段

## 安全要求

- 真实密钥不得进入仓库。
- 真实密钥不得进入配置文件。
- 真实密钥不得进入 debug API 响应。
- 真实密钥不得进入 debug 页面。
- 真实密钥不得进入结构化日志。
- 错误信息中不得包含密钥内容。

允许展示：

- `secret_ref`
- `secret_status`
- 失败原因摘要

## 部署建议

部署时应包含三类文件：

1. 服务二进制
2. 业务配置文件
3. 服务器 secret 目录

推荐布局：

- `/opt/grid-trade/trader`
- `/opt/grid-trade/configs/example.yaml`
- `/etc/grid-trade/accounts/*.json`

推荐与 `systemd` 配合：

- `systemd` 负责启动服务
- 主配置通过 `TRADER_CONFIG` 指向部署目录
- `secret_dir` 由配置指定

## 实现顺序

建议按以下顺序改造：

1. 修改配置结构，新增 `enabled` 和 `secret_ref`
2. 新增 `SecretProvider` 抽象
3. 实现 `FileSecretProvider`
4. 调整启动流程，按账号加载 secret
5. 扩展 gateway/debug 状态模型
6. 更新 debug console 页面
7. 更新部署文档和示例配置

## 第一版验收标准

- 服务能从配置文件读取多账号列表。
- 服务能从 `system.secret_dir` 读取单账号 secret 文件。
- secret 文件缺失时，该账号显示为 `degraded`。
- 其他账号仍能正常启动。
- debug console 能显示 `secretRef` 和 `secretStatus`。
- debug console 不显示任何真实密钥内容。
- `go test ./...` 通过。

## 延后事项

- 热更新账号列表
- 云 Secret Manager
- 数据库存账号
- 自动密钥轮换
- 网页端账号管理
