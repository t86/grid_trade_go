# Binance WebSocket 交易服务设计

## 概述

本文档定义第一版面向生产环境的 Binance 交易服务设计。系统以 WebSocket 优先为原则，尽量减少对 REST 的依赖，从而降低限频压力。服务目标是支持 Binance 现货和 USD-M 合约的实盘交易，其中合约账户按双向持仓模式运行。整体框架定位为通用策略引擎，网格策略作为第一个插件策略接入。

## 目标

- 构建一套基于 Go 的 Binance 现货和 USD-M 合约实盘交易服务。
- 在支持的能力范围内，优先使用 WebSocket 处理行情、账户事件和交易请求。
- 将 REST 限制在冷启动、状态对账和受控兜底场景。
- 在接入交易所前统一完成限频、风控和订单准入控制。
- 通过稳定的内部接口支持插件式策略扩展。
- 在不将网格逻辑耦合进核心内核的前提下，交付第一版网格策略。

## 非目标

- 第一版不支持多交易所。
- 第一版不采用微服务分布式部署。
- 第一版不支持模拟盘或纸面交易。
- 第一版不同时支持多类策略家族。
- 第一版不建设完整深度行情基础设施，除非后续策略确实需要。

## 设计原则

- 策略代码永远不直接调用 Binance API。
- 现货和合约在核心层共享统一抽象，交易所差异隔离在适配器层。
- WebSocket 是默认运行路径，REST 只用于恢复和初始化。
- 系统状态只能根据交易所确认事件或用户流事件推进，不能基于本地猜测推进。
- 合约持仓必须按 `symbol + positionSide` 建模，以安全支持双向持仓。
- 系统优先选择保守降级，而不是乐观重试。

## 总体架构

第一版建议实现为单个 Go 二进制服务，但内部边界要足够清晰，使后续可以自然拆分为两个逻辑服务：交易网关和策略引擎。

核心层如下：

- `Exchange Gateway`
  管理 Binance 现货和合约的会话、用户流、WS API 请求，以及受控的 REST 兜底调用。
- `Rate Limit Governor`
  跟踪请求额度、连接频率以及本地退避状态。
- `State Store`
  保存标准化后的行情、余额、持仓、未成交订单和系统健康状态。
- `Order Admission Gate`
  在订单到达交易所适配器前执行统一准入校验。
- `Risk Engine`
  执行账户级、策略级、标的级和方向级硬风控。
- `Strategy Runtime`
  负责加载策略插件、向策略分发标准化事件，并接收策略返回的订单意图。

高层数据流：

`Binance WS/REST -> Gateway -> 标准化事件 -> State Store / Strategy Runtime -> Order Intent -> Admission Gate -> Risk Engine -> Exchange Adapter -> Binance`

## 模块布局

建议的代码目录结构：

- `cmd/trader`
  进程入口和依赖装配。
- `internal/kernel/bus`
  内部事件总线，负责市场、账户、订单和控制事件的传递。
- `internal/gateway`
  负责适配器调度、连接管理、对账和交易所出入站流量协调。
- `internal/exchange/binance/spot`
  现货适配器，处理现货专有报文标准化和下单翻译。
- `internal/exchange/binance/futures`
  合约适配器，处理合约专有报文标准化以及双向持仓逻辑。
- `internal/admission`
  负责订单参数校验、幂等校验、交易所规则校验和额度校验。
- `internal/risk`
  负责硬风控和 Kill Switch 控制。
- `internal/state`
  负责规范化后的内存状态存储和快照读取。
- `internal/strategy`
  负责策略运行时和插件接口定义。
- `internal/strategy/grid`
  第一版网格策略实现。
- `internal/config`
  负责配置解析和配置校验。
- `internal/logger`
  负责结构化日志和审计辅助能力。

## 领域模型

核心标准化类型：

- `MarketType`
  取值：`Spot`、`FuturesUM`。
- `InstrumentKey`
  字段：`market`、`symbol`。
- `PositionKey`
  字段：`market`、`symbol`、`positionSide`。
- `OrderIntent`
  字段：`strategyID`、`market`、`symbol`、`positionSide`、`side`、`orderType`、`price`、`quantity`、`timeInForce`、`reduceOnly`、`clientOrderID`、`reason`。
- `OrderRecord`
  保存本地下单标识、交易所订单标识、订单状态、成交数量、均价、来源策略和时间戳。
- `AccountSnapshot`
  保存余额、持仓、未成交订单和更新时间。
- `MarketEvent`
  表示标准化后的 `bookTicker`、`trade`、`kline` 和 `markPrice` 事件。
- `OrderEvent`
  表示标准化后的订单生命周期事件。
- `AccountEvent`
  表示标准化后的余额和持仓更新事件。

## WebSocket 优先执行模型

### 连接分类

网关层维护三类逻辑通道：

- `Market Data WS`
  订阅低成本行情流。第一版网格策略使用 `bookTicker` 即可。
- `User Data WS`
  分别维护现货和合约用户事件流。
- `Order Channel`
  在 Binance 支持的前提下，优先通过 WebSocket API 发单和撤单。

所有连接统一交给 `Reconnect Supervisor` 管理，必须具备：

- 指数退避
- 随机抖动
- 最大重连窗口
- 集中的连接状态跟踪

每个会话都有明确生命周期状态：

- `connecting`
- `active`
- `degraded`
- `backing_off`
- `banned_until`

### 冷启动

启动流程：

1. 加载配置和交易所元数据。
2. 通过 REST 拉取初始余额、持仓、未成交订单和交易规则快照。
3. 初始化本地状态。
4. 建立用户流和行情流。
5. 只有在用户流确认就绪后，才允许交易开启。

### 增量同步

运行过程中：

- 行情通过 WebSocket 增量更新本地市场状态。
- 用户流负责推进订单、余额和持仓状态。
- 策略层统一从 `State Store` 读取状态，不维护独立真相。

### 恢复流程

当连接中断或系统怀疑状态不一致时：

1. 将网关标记为 `degraded`。
2. 停止新的开仓订单，必要时仅允许减仓单。
3. 使用统一退避策略重建连接。
4. 通过受控 REST 快照执行对账。
5. 重建本地真实状态。
6. 只有在对账成功后，才恢复正常交易。

## 订单生命周期

标准下单链路：

`Strategy -> OrderIntent -> Admission -> Risk -> Exchange Adapter -> Binance -> Ack / Event -> State Store`

必须满足以下规则：

- 本地提交成功不等于订单成功。
- 策略状态只能依据交易所确认或用户流事件推进。
- 重试只能由中心化组件统一控制，策略层不能自行决定。
- `clientOrderID` 必须具备足够稳定性，以支持幂等和回放追踪。

## 限频设计

系统必须假设 Binance 公布的规则只覆盖了部分有效约束，因此本地保护必须保守。

必须具备的限频器：

- `Connection Limit`
  按 IP、市场和通道控制 WebSocket 建连和重连频率。
- `Request Weight Limit`
  跟踪所有 WS API 和兜底 REST 请求的本地权重预算。
- `Order Rate Limit`
  在 Binance 拒绝之前，本地先控制下单、撤单和未成交订单压力。
- `Strategy Quota`
  为每个策略划分订单能力配额，防止单策略拖垮全局账户。

限频行为要求：

- 收到 `429` 后立即进入退避，并优先遵循交易所返回的重试时间。
- 收到 `418` 后冻结相关流量，直到已知封禁窗口结束。
- 限频状态变化必须可以从健康检查、日志和指标中观察到。

## 风控设计

第一版只实现硬风控，不做复杂组合风险模型。

账户级：

- 最大总敞口
- 最大保证金使用率
- 最大日内亏损
- 最大未成交订单数

策略级：

- 单策略最大敞口
- 单策略最大挂单数
- 单笔最大名义价值
- 连续失败熔断

标的级：

- 单标的最大敞口
- 双边挂单最大总量
- 对需要保护的策略设置最小价差或最小距离限制

合约双向持仓方向级：

- 按 `symbol + positionSide` 独立计算限制

## 订单准入流水线

每一笔 `OrderIntent` 都必须经过同一条流水线：

1. 参数校验
2. 幂等校验
3. 交易所规则校验
4. 限频校验
5. 风控校验
6. 发往交易所

这个顺序必须固定，以保证拒单行为可审计且可预测。

## 故障处理

系统应当优先安全降级，而不是激进自恢复。

必须具备的行为：

- 收到 `429` 时，立即停止对应请求流并进入退避。
- 收到 `418` 时，冻结相关流量直到交易所封禁结束。
- 用户流中断时，停止新的开仓订单。
- 行情流中断时，暂停依赖实时价格的策略。
- 状态不一致时，进入对账流程，完成后才能恢复。
- 交易所连续返回未知错误时，仅允许有限次带抖动重试，之后进入关闭保护状态。

系统必须支持 `Kill Switch`：

- 全局停单
- 单策略停单
- 只减仓模式
- 明确的人工恢复动作

## 策略运行时

策略必须通过稳定插件接口接入。

示例接口如下：

```go
type Strategy interface {
    ID() string
    Markets() []MarketBinding
    OnStart(ctx context.Context, env RuntimeEnv) error
    OnStop(ctx context.Context) error
    OnMarketEvent(ctx context.Context, event MarketEvent) []OrderIntent
    OnOrderEvent(ctx context.Context, event OrderEvent) []OrderIntent
    OnAccountEvent(ctx context.Context, event AccountEvent) []OrderIntent
    Snapshot() StrategyState
}
```

运行时规则：

- 策略只消费标准化事件。
- 策略只输出 `OrderIntent`，不能直接调用交易接口。
- 每个策略实例应串行处理事件，避免策略内部并发污染。
- 策略状态必须支持快照，以支持重启恢复和审计。

`RuntimeEnv` 至少应提供：

- 标准化状态读取能力
- 交易规则和 symbol 元数据查询能力
- 结构化日志和指标能力
- 策略配置读取能力
- 时钟和调度器能力
- 策略私有状态持久化钩子

## 网格策略插件

网格策略是第一版插件，但不应反向塑造核心架构。

配置项：

- `market`
- `symbol`
- `mode`
- `gridCount`
- `upperPrice`
- `lowerPrice`
- `baseOrderSize`
- `takeProfitMode`
- `hedgeModeBehavior`
- `maxInventory`
- `maxActiveOrders`

内部状态：

- 当前网格区间
- 各网格目标挂单
- 当前活动订单映射
- 按方向维护的库存摘要
- 暂停或只减仓状态
- 上次重建网格时间

网格状态机：

- `Bootstrapping`
- `SeedingGrid`
- `Active`
- `Rebalancing`
- `ReduceOnly`
- `Paused`
- `Recovering`

对于合约双向持仓，第一版应将多头网格和空头网格视为两套逻辑独立的子网格，各自维护订单和库存上限。

## 配置设计

配置建议分成三层：

- `system config`
  包含日志、端点、全局限频、全局风控默认值和功能开关。
- `account config`
  包含账户启用状态、允许市场、杠杆策略和交易标的白名单。
- `strategy config`
  包含每个策略实例自己的参数和运行开关。

第一版建议使用 YAML。密钥等敏感信息应来自环境变量或外部密钥注入，而不是写入仓库明文。

## 可观测性

第一版至少需要以下输出：

- `structured logs`
  记录连接状态变化、订单准入决策、风控拒绝和交易所响应。
- `metrics`
  暴露会话状态、事件吞吐、下单成功率、拒单数、`429`、`418`、敞口和策略级收益等指标。
- `audit trail`
  能够将一笔订单意图从准入到派发、交易所响应和最终状态串联追踪。
- `health endpoints`
  暴露 readiness、liveness，以及 `enabled`、`degraded`、`reduce_only`、`halted` 等交易状态。

## 测试策略

第一版需要至少四层测试：

- `unit tests`
  覆盖准入、风控、限频器和网格状态机。
- `adapter contract tests`
  使用录制报文或示例报文，验证现货和合约适配器的标准化正确性。
- `integration tests`
  在本地运行时中注入行情、订单和断线事件，验证完整状态流转。
- `replay tests`
  回放真实事件序列，验证系统恢复能力和状态一致性。

## 第一版交付范围

第一版应交付：

- 一个 Go 二进制服务
- 清晰的内部模块边界：gateway、admission、risk、state、runtime 和 Binance adapters
- 现货和 USD-M 合约支持
- 合约双向持仓支持
- WebSocket 优先交易链路
- 受控的 REST 初始化和对账补偿
- 插件式策略运行时
- 一个网格策略插件
- 本地限频、Kill Switch、只减仓模式和对账恢复流程

## 延后事项

以下内容明确延后：

- 多进程或分布式部署
- 模拟盘环境
- 多交易所抽象
- 复杂组合风控模型
- 高级深度行情策略
- 大规模策略目录

## 外部参考

- Binance Spot WebSocket API 限频说明
- Binance Spot 未成交订单计数规则
- Binance USD-M Futures WebSocket API 通用说明

以上资料用于支撑本文中的限频和退避设计假设，但系统仍应保持保守行为，因为交易所可能存在未公开的附加风控或封控规则。
