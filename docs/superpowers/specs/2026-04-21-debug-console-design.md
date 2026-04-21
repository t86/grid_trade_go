# Binance 交易服务调试控制台设计

## 概述

本文档定义第一版内置调试控制台的设计。控制台服务于 Binance WebSocket 交易服务，目标是让开发者和运维人员能直观看到 Spot 与 Futures 的连接健康状态、账户用户流状态、告警状态和交易保护状态。

第一版调试控制台只做只读运维看板，不提供任何交易操作按钮，不允许在页面内开仓、撤单、恢复交易或关闭风控。

## 目标

- 提供一个内置在 Go 交易服务中的本地调试页面。
- 首页按市场分组展示 `Spot` 和 `Futures` 两个主面板。
- 顶部提供混合告警带，同时展示连接异常和交易保护告警。
- 让用户能在 3 秒内判断哪个市场有问题、问题严重程度、影响账户和问题类别。
- 支持展开账户明细，查看 listen key、用户流、心跳、最近错误和保护状态。
- 为后续交易链路调试、策略状态可视化和事件时间线预留扩展边界。

## 非目标

- 第一版不做交易操作按钮。
- 第一版不做登录、权限和用户管理。
- 第一版不做复杂图表、趋势曲线或实时 K 线。
- 第一版不做二级路由。
- 第一版不展示完整原始 JSON 面板。
- 第一版不做独立前端项目。

## 设计原则

- 调试控制台是运维观测面板，不是交易终端。
- 页面默认只读，不能绕过风控或准入层。
- 前端只消费展示模型，不直接绑定内部 gateway、risk、limit 等运行时结构。
- 首屏优先展示问题和影响范围，而不是展示所有细节。
- 告警状态必须高于普通健康指标展示。
- Spot 和 Futures 的视觉结构保持一致，降低理解成本。

## 产品形态

调试控制台采用内置运维控制台方案：

- Go 服务提供 `/debug/*` API。
- Go 服务同时托管 `web/debug` 下的静态页面。
- 页面通过浏览器访问，不需要独立前端构建链路。
- 第一版使用原生 HTML、CSS 和 JavaScript。

选择该方案的原因：

- 部署最简单，和交易服务同进程启动。
- 本地调试成本低，不需要额外起前端 dev server。
- 第一版需求明确，不需要引入复杂前端框架。
- 后续如果界面复杂度上升，可以再拆成独立前端。

## 首页结构

首页采用已确认的 `方案 B`：顶部告警带 + Spot/Futures 双市场面板。

页面从上到下分为三块：

1. `混合告警带`
   显示连接异常和交易保护告警，按严重度排序。
2. `市场健康卡片`
   左侧为 Spot，右侧为 Futures，展示中等密度的健康摘要。
3. `账户明细展开区`
   每个市场卡片可展开账户列表，展示具体连接和保护状态。

## 告警带设计

顶部告警带同时包含两类告警：

- `connection`
  包括 WebSocket 断线、用户流心跳超时、listen key 失效、重连退避、REST listen key 获取失败。
- `protection`
  包括 reduce-only 生效、kill switch 生效、限频退避、风控拒绝、交易暂停。

告警按以下优先级展示：

1. `critical`
2. `warning`
3. `info`

每条告警默认展示：

- 严重度
- 市场
- 账户
- 标题
- 当前状态
- 发生时间

点击告警后展开：

- 详细说明
- 最近更新时间
- 关联状态，例如 `reduce_only`、`backoff_active`、`listen_key_expiring`

## 市场健康卡片

首页包含两个主卡片：

- `Spot`
- `Futures`

每个卡片采用中等信息密度，默认展示：

- `health`
  取值为 `healthy`、`warning`、`degraded`。
- `onlineAccounts`
  当前在线账户数。
- `degradedAccounts`
  当前降级账户数。
- `avgHeartbeatLagMs`
  用户流平均心跳延迟。
- `listenKeyHealthyAccounts`
  listen key 正常账户数。
- `lastReconnectAt`
  最近一次重连时间。
- `reduceOnlyAccounts`
  当前处于只减仓的账户数。
- `lastError`
  最近错误摘要。

状态颜色：

- `healthy` 使用绿色。
- `warning` 使用琥珀色。
- `degraded` 和 `critical` 使用红色。
- `reduce-only` 使用橙色标签。
- `kill switch` 使用红色实心标签。

## 账户明细行

市场卡片展开后显示账户明细。账户行按问题优先级排序：

1. `degraded`
2. `reduce-only`
3. `backoff active`
4. `healthy`

账户行字段：

- `account`
- `sessionState`
- `userStreamState`
- `listenKeyState`
- `listenKeyExpiresAt`
- `lastHeartbeatAt`
- `lastReconnectAt`
- `reduceOnly`
- `killSwitch`
- `activeBackoff`
- `lastError`

## 展示模型

后端对前端暴露展示模型，不直接暴露内部结构。

### AlertSummary

字段：

- `id`
- `severity`
- `category`
- `market`
- `account`
- `title`
- `detail`
- `since`
- `status`

约束：

- `severity` 取值为 `info`、`warning`、`critical`。
- `category` 取值为 `connection`、`protection`。
- `market` 取值为 `spot`、`futures_um`、`system`。
- `status` 取值为 `active`、`recovered`。

### MarketHealthCard

字段：

- `market`
- `health`
- `onlineAccounts`
- `degradedAccounts`
- `avgHeartbeatLagMs`
- `lastReconnectAt`
- `reduceOnlyAccounts`
- `listenKeyHealthyAccounts`
- `lastError`

约束：

- `market` 取值为 `spot`、`futures_um`。
- `health` 取值为 `healthy`、`warning`、`degraded`。

### AccountConnectionRow

字段：

- `account`
- `sessionState`
- `userStreamState`
- `listenKeyState`
- `listenKeyExpiresAt`
- `lastHeartbeatAt`
- `lastReconnectAt`
- `reduceOnly`
- `killSwitch`
- `activeBackoff`
- `lastError`

## 后端接口

第一版提供三个只读接口。

### GET /debug/dashboard

用途：

- 首页首屏数据源。
- 返回顶部告警和两个市场健康卡片。

响应结构：

```json
{
  "alerts": [],
  "markets": []
}
```

### GET /debug/accounts?market=spot|futures_um

用途：

- 返回指定市场下的账户连接明细。
- 用于市场卡片展开区。

响应结构：

```json
{
  "market": "futures_um",
  "accounts": []
}
```

### GET /debug/events?market=...&account=...

用途：

- 返回最近连接事件和保护事件。
- 第一版可以先服务于接口验证，页面不必默认展示。

响应结构：

```json
{
  "events": []
}
```

## 文件结构

建议新增文件：

- `internal/debugview/types.go`
  定义 `AlertSummary`、`MarketHealthCard`、`AccountConnectionRow` 和 dashboard 响应结构。
- `internal/debugview/service.go`
  聚合 gateway、限频、风控、策略运行时等状态，产出展示模型。
- `internal/debugview/service_test.go`
  测试告警汇总、市场健康聚合和账户排序。
- `internal/debughttp/handler.go`
  暴露 `/debug/dashboard`、`/debug/accounts` 和 `/debug/events`。
- `internal/debughttp/handler_test.go`
  测试接口状态码、响应结构和参数校验。
- `web/debug/index.html`
  调试控制台首页。
- `web/debug/app.css`
  页面样式。
- `web/debug/app.js`
  拉取 debug API 并渲染告警带、市场卡片和账户展开区。

需要修改的文件：

- `cmd/trader/main.go`
  挂载 debug HTTP handler 和静态页面。
- `internal/gateway/service.go`
  暴露只读状态快照，供 debugview 聚合使用。

## 数据来源

第一版至少从 gateway 提供以下状态：

- 每个账户每个市场的 session state。
- 最近重连时间。
- 最近错误。
- reduce-only 状态。
- listen key 是否健康。
- listen key 最近刷新时间。
- listen key 最近失败时间。
- 最近心跳时间。
- 当前 heartbeat lag。
- backoff 是否生效。

后续可以接入：

- rate limit governor 状态。
- risk engine 拒绝记录。
- order admission 拒绝原因。
- strategy runtime 当前策略状态。

## 可观测性要求

调试控制台本身也需要可观测：

- API 返回错误时，页面显示错误条，而不是空白。
- 数据为空时显示空状态，例如 `No active alerts`。
- 前端轮询失败时显示最近成功刷新时间。
- 页面刷新频率第一版固定为 2 秒。
- 页面不得因为某个字段缺失而整体崩溃。

## 安全边界

第一版只读：

- 不提供下单。
- 不提供撤单。
- 不提供恢复交易。
- 不提供关闭 kill switch。
- 不显示 API secret。
- listen key 默认不完整展示，只显示状态或尾号摘要。

如果未来需要操作能力，必须单独设计权限、审计和确认流程。

## 实现顺序

建议按以下顺序实现：

1. 新增 `debugview` 展示模型和聚合服务。
2. 给 gateway 增加只读状态快照。
3. 新增 `/debug/dashboard` 和 `/debug/accounts` API。
4. 新增静态页面并挂载到 Go 服务。
5. 用假数据验证页面布局。
6. 用真实 gateway 状态验证接口和页面。
7. 补 `/debug/events`，用于后续事件时间线。

## 第一版验收标准

- 访问 `/debug/` 能看到调试控制台首页。
- 首页顶部能显示混合告警。
- 首页能同时显示 Spot 和 Futures 健康卡片。
- 市场卡片能展示中等密度指标。
- 市场卡片能展开账户明细。
- `/debug/dashboard`、`/debug/accounts`、`/debug/events` 均返回 JSON。
- `go test ./...` 通过。
- 页面不包含任何交易操作按钮。

## 延后事项

- 独立前端工程。
- 登录认证。
- 操作按钮。
- 实时图表。
- 原始 JSON 查看器。
- 策略行为详情页。
- 订单链路时间线。
- WebSocket 推送式前端刷新。
