# CrawlerGo CDP 协议兼容性设计

## 状态

待实施。本文件仅记录后续调查与实施方案，不代表已修改依赖或浏览器版本。

## 背景

CrawlerGo 当前依赖 `chromedp v0.8.2` 和 `cdproto v0.0.0-20220629234738-4cfc9cdeeb92`，`go.mod` 声明 Go 1.16。近期运行出现以下日志：

- `could not unmarshal event: unknown IPAddressSpace value`
- `could not unmarshal event: unknown ClientNavigationReason value`

`cdproto` 将 Chrome DevTools Protocol（CDP）事件解码为生成的 Go 类型。当前 CDP 中 `IPAddressSpace` 包含 `Loopback`，`ClientNavigationReason` 包含 `initialFrameNavigation`、`other` 等值；当前锁定的旧版 `cdproto` 不认识这些新增值，因此拒绝解码对应事件。错误日志没有打印实际枚举值，实施前仍需确认运行环境中的 Chrome 版本及实际触发值。

chromedp 收到不可解码事件时记录错误并跳过该事件，事件循环继续运行。CrawlerGo 当前代码没有直接引用这两个枚举字段。验收时任务仍正常结束并生成结果文件，因此这些告警不是任务终止或未生成聚合 JSON 的直接原因；但对应 CDP 事件会丢失，依赖这些事件数据的功能可能受影响。

## 目标

- 明确 CrawlerGo 支持的 Chrome、chromedp、cdproto 与 Go 版本组合。
- 使受支持的 Chrome 版本不再触发上述枚举解码错误。
- 保持多目标、Scope、robots、fuzz 和聚合结果行为不变。
- 不通过隐藏日志或忽略错误来假装事件已成功解码。

## 兼容方案

### 方案 A：升级 chromedp 与 cdproto

成套选择能解码部署 Chrome 所发出 CDP 事件的 `chromedp`、`cdproto` 版本，并评估它们对 Go 版本及现有浏览器任务 API 的要求。

优点：能够继续使用部署环境中的受支持 Chrome，并跟进 CDP 变化。

风险：依赖升级可能带来 Go 基线提高、公开 API 或事件处理行为变化；当前 CrawlerGo 声明 Go 1.16，不能直接将 `cdproto` 升至主线最新版而不先评估其 Go 版本要求。

### 方案 B：固定 Chrome 版本

保留现有 chromedp/cdproto 依赖，将运行 Chrome 固定到与当前 CDP 定义相匹配的版本，并在启动或部署文档中明确该版本要求。

优点：CrawlerGo 代码改动相对较少。

风险：生产环境需要持续管理 Chrome 安全更新；如果 Chrome 由部署环境提供，版本固定和跨平台交付需要额外维护。

### 推荐决策顺序

先确认 Tanscan 实际启动的 Chrome 可执行文件版本，并评估可保持现有 Go 基线的 chromedp/cdproto 版本组合。若没有合适组合，再比较升级 Go 基线与固定 Chrome 的部署影响，确定方案后实施。不能仅凭可配置的 User-Agent 判断 CDP 版本，CDP 协议取决于实际启动的 Chrome 可执行文件。

## 实施步骤

1. 收集部署环境中实际 Chrome 路径和版本；用不含 URL、请求头等敏感数据的诊断方式确认触发错误的 CDP 事件及枚举值。
2. 核对候选 chromedp/cdproto 版本与 Go 版本、Windows/Linux 构建及当前 CrawlerGo 浏览器池调用的兼容性。
3. 选定方案并更新依赖或浏览器版本约束；如升级依赖，优先成套升级 chromedp 与 cdproto。
4. 更新相关测试、构建命令或部署说明，使支持的版本组合清晰可检查。
5. 重建随 Tanscan 发布的 Windows/Linux CrawlerGo 二进制。

## 验收标准

- 在声明支持的 Chrome 版本上运行真实浏览器任务，不再出现上述两种枚举解码错误。
- CrawlerGo 完整测试 `go test ./...` 通过，Windows 与 Linux amd64 构建通过。
- 多 URL 输入仍使用一个 CrawlerGo 进程并生成一份聚合结果。
- `curr_domain`、`wild_domain`、`unlimit` 范围行为不变。
- 开启 robots 和 fuzz 后，入口及发现结果符合预期，任务可正常结束。
- 对比改动前后的验收夹具，确认结果集合没有非预期变化。

## 非目标与限制

- 不修改 Tanscan API、数据库结构或持久化数据。
- 不增加 CrawlerGo 侧 SSRF 检查。
- 不仅为消除日志而吞掉 CDP 解码错误。
- DNS 或网络出口隔离不属于本方案。
