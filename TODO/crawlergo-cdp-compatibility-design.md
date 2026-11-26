# CrawlerGo CDP 协议兼容性设计

## 状态

方案 A 已在本仓库实施并通过本机 Chrome/WebSocket 验证。生产环境支持的 Chrome 版本范围仍需按实际部署环境确认。

- 依赖：`chromedp v0.14.2`、`cdproto v0.0.0-20250724212937-08a3db8b4327`。
- Go 基线：`go.mod` 声明 Go 1.24（满足依赖最低要求）；Docker、CI 和发布工作流使用受支持的 Go 1.27.4。
- 本机验证：Chrome 156.0.8078.80，CDP protocol 1.3；经本地 WebSocket 爬取本机 HTTP 页面，观察到并成功解码 `IPAddressSpace=Loopback`。
- CLI 验证：通过 `--chrome-ws-url` 完成本地端到端任务，覆盖多目标聚合、robots/fuzz 路径、`curr_domain` 范围、站外请求观测及 POST 正文采集；修复 fetch hook 丢弃 `init` 参数的问题后通过。
- Docker：运行镜像已迁移到 Ubuntu 24.04；Chrome for Testing 固定为 158.0.8127.88，并校验归档 SHA-256。临时手动 Docker 工作流已成功构建镜像，并通过容器内 Chrome 对本机 HTTP fixture 执行 CLI WebSocket 集成测试（Actions run 45802160235）；验证后已移除该工作流。常规 Go CI 仍运行构建和测试。
- `ClientNavigationReason` 的 `initialFrameNavigation`、`other` 由事件载荷回归测试覆盖；本机任务未声称一定会触发这两个值。
- 待部署环境确认：当前工作区没有生产 Chrome 清单，因此生产版本矩阵仍需部署环境验证。本机没有 Docker/Podman，容器镜像验证由 GitHub Actions 完成。

## 背景

升级前 CrawlerGo 依赖 `chromedp v0.8.2` 和 `cdproto v0.0.0-20220629234738-4cfc9cdeeb92`，`go.mod` 声明 Go 1.16。运行曾出现以下日志：

- `could not unmarshal event: unknown IPAddressSpace value`
- `could not unmarshal event: unknown ClientNavigationReason value`

`cdproto` 将 Chrome DevTools Protocol（CDP）事件解码为生成的 Go 类型。当前 CDP 中 `IPAddressSpace` 包含 `Loopback`，`ClientNavigationReason` 包含 `initialFrameNavigation`、`other` 等值；当前锁定的旧版 `cdproto` 不认识这些新增值，因此拒绝解码对应事件。原错误日志没有打印实际枚举值；本次读取并验证了开发机 Chrome 156.0.8078.80，生产环境触发该错误时使用的浏览器版本仍需由部署环境确认。

CrawlerGo 支持两种浏览器连接方式：通过 `--chromium-path` 启动本地 Chrome，以及通过 `--chrome-ws-url` 连接已有浏览器。本方案以本机 Chrome 的本地 WebSocket 连接作为主要集成验证路径，避免把远端浏览器或外网目标引入诊断结果。通过已建立的 CDP 会话读取浏览器产品版本和协议版本；诊断记录不得包含完整 WebSocket 地址、目标 URL 或请求头等敏感信息。

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

风险：依赖升级可能带来 Go 基线提高、公开 API 或事件处理行为变化。本次选定版本最低要求 Go 1.24；Docker、CI 和发布工作流改用受支持的 Go 1.27.4，后续升级仍需核对依赖的 Go 要求与生成协议变化。

### 方案 B：固定 Chrome 版本

保留现有 chromedp/cdproto 依赖，将运行 Chrome 固定到与当前 CDP 定义相匹配的版本，并在启动或部署文档中明确该版本要求。

优点：CrawlerGo 代码改动相对较少。

风险：生产环境需要持续管理 Chrome 安全更新；如果 Chrome 由部署环境提供，跨平台交付需要额外维护。当前 Docker 镜像固定 Chrome for Testing 版本和 SHA-256，后续更新必须同时升级版本、校验值并运行兼容性测试；外部提供或通过 WebSocket 连接的 Chrome 仍需单独声明版本范围。

### 推荐决策顺序

本次采用方案 A：升级 chromedp/cdproto，并把 Go 基线明确为 1.24。开发机本地 Chrome/WebSocket 验证用于复现和回归；生产环境支持范围仍需从实际连接的浏览器会话读取版本信息后另行声明。不能仅凭可配置的 User-Agent 判断 CDP 版本。

## 实施步骤

1. 已在开发机启动本地 Chrome，绑定 `127.0.0.1` 调试地址并使用独立用户目录；通过 `--chrome-ws-url` 连接，并只记录浏览器产品版本和 CDP protocol，不记录完整 WebSocket 地址。
2. 已使用本机 HTTP 测试页验证 WebSocket 导航、Loopback 响应事件及 CLI 结果；CLI 回归覆盖多目标聚合、robots/fuzz、范围过滤、浏览器观测请求和 POST 正文。新增 `ClientNavigationReason` 事件载荷测试，覆盖 `initialFrameNavigation` 和 `other`。
3. 已升级 chromedp/cdproto，并修复新 `Network.Request` 用 `PostDataEntries` 表示请求体所需的兼容处理；已将 Go 构建工具链同步至受支持版本。
4. 已运行完整 Go 测试及 Windows/Linux amd64 构建。临时手动 Docker 工作流已通过镜像构建和容器内 Chrome 本地 WebSocket 集成回归（Actions run 45802160235），随后已移除；常规 Go CI 不自动构建 Docker 镜像。
5. 生产环境 Chrome 支持范围和发布产物重建仍待部署环境验证。当前工作区不含生产 Chrome 清单；本机 CDP 集成验证仅覆盖 Chrome 156.0.8078.80，本地 Docker 镜像固定 Chrome for Testing 158.0.8127.88。

本机可通过 Chrome 的 `--remote-debugging-port` 启动参数开启本地调试端口，然后用 PowerShell 动态读取 WebSocket 地址并运行集成测试。测试会拒绝非回环地址，且只输出 Chrome 产品版本与 CDP protocol：

```powershell
$chromePath = Join-Path $env:LOCALAPPDATA 'Google\Chrome\Application\chrome.exe'
$profilePath = Join-Path $env:TEMP 'crawlergo-cdp-profile'
Start-Process -FilePath $chromePath -WindowStyle Hidden -ArgumentList @(
  '--headless=new', '--disable-gpu', '--remote-debugging-address=127.0.0.1',
  '--remote-debugging-port=9222', "--user-data-dir=$profilePath"
)
$env:CRAWLERGO_CHROME_WS_URL = (Invoke-RestMethod 'http://127.0.0.1:9222/json/version').webSocketDebuggerUrl
go test ./pkg/engine ./cmd/crawlergo -run 'Test(LocalChromeWebSocket|CLIWithLocalChromeWebSocket)' -count=1 -v
Remove-Item Env:CRAWLERGO_CHROME_WS_URL
```

## 验收标准

- 使用本机 Chrome、本地 WebSocket 地址和本机 HTTP 测试页完成真实浏览器任务；验证连接、页面访问和 Loopback 响应事件解码正常，且未出现上述枚举解码错误。
- 对本机 Chrome 未实际触发的枚举，使用对应 CDP 事件载荷回归验证解码成功；未覆盖的枚举不得以“日志未出现”作为通过依据。
- 验收记录包含浏览器产品版本和 CDP 协议版本，且不包含完整 WebSocket 地址、目标 URL 或请求头。
- 明确生产支持的 Chrome 版本范围；本机验证只代表 Chrome 156.0.8078.80 及本地 WebSocket 连接路径。
- CrawlerGo 完整测试 `go test ./...` 通过，Windows 与 Linux amd64 构建通过。
- 多 URL 输入仍使用一个 CrawlerGo 进程并生成一份聚合结果。
- `curr_domain`、`wild_domain`、`unlimit` 范围行为不变。
- 开启 robots 和 fuzz 后，入口及发现结果符合预期，任务可正常结束。
- 对比改动前后的验收夹具，确认结果集合没有非预期变化。

## 非目标与限制

- 不增加 CrawlerGo 侧 SSRF 检查。
- 不仅为消除日志而吞掉 CDP 解码错误。
- DNS 或网络出口隔离不属于本方案。
