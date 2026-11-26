# CrawlerGo 多目标与 Scope 改造设计

## 背景

Tanscan 的一次扫描任务可以包含多个获准 URL，但当前 CrawlerGo 适配器按 URL 循环启动进程并分别读取结果。CrawlerGo 自身又以第一个 URL 的 Host 作为唯一范围，导致多个入口无法在同一浏览器任务中共同发现资源，也无法得到一份任务级聚合结果。

## 目标

- 一个 Tanscan 爬虫任务启动一个 CrawlerGo 进程，并把任务中的全部入口 URL 一次传入。
- CrawlerGo 在整个任务中共享浏览器、并发池、去重集合和爬取数量上限。
- CrawlerGo 的队列和 `req_list` 按任务 Scope 收集；`all_req_list` 继续保留观察到的全部请求。
- 一个进程只生成并读取一个聚合 JSON 结果文件。
- 保持现有数据库和 Tanscan API 不变。

## Scope 规则

新增 `--scope-filter`，支持与 Tanscan 一致的三种值：

| 值 | 允许进入发现队列和 `req_list` 的 Host |
| --- | --- |
| `curr_domain` | 本任务全部入口 URL 的精确 Host 集合 |
| `wild_domain` | 本任务入口的注册根域，以及各根域下的子域 |
| `unlimit` | 任意 HTTP 或 HTTPS Host |

未指定时采用 `curr_domain`。无效值应在创建任务时返回错误。范围检查应用于入口 URL、robots.txt 和 fuzz 生成的 URL、重定向及页面发现的请求。浏览器页面中观察到的范围外资源仍可以作为浏览器加载请求出现于 `all_req_list`，但不会进入爬取队列或 `req_list`。此改造不增加 CrawlerGo 的 SSRF 检查；Tanscan 现有的任务校验和爬取后范围校验保持不变。

## 任务处理

1. Tanscan 将所有任务入口和 Scope 参数组装到同一组 CrawlerGo 命令行参数中。
2. CrawlerGo 基于全部入口创建一个 Scope filter；入口间允许互相发现，所有入口共同消耗任务级爬取上限。
3. robots.txt 和 fuzz 按唯一的 `scheme://host` 入口分别扩展一次，扩展结果仍通过同一 Scope filter。
4. `req_list` 输出范围内、去重且通过现有静态资源/智能过滤策略的请求；`all_req_list` 输出去重后的全部观测请求。
5. 子域名统计按任务全部入口的注册根域合并，保留旧的单根域调用接口。
6. Tanscan 为本次进程指定一个任务级结果文件，读取一次聚合结果，再执行现有任务 Scope、目标及 SSRF 后置校验。

单入口任务继续保留 CrawlerGo 现有的反向 Scheme 入口兼容行为。Tanscan 最终范围校验仍决定哪些结果可用于扫描。

## 兼容与实施边界

- 保留现有 `NewSimpleFilter(host)` 和单根域统计调用接口，新增按入口集合创建 filter 的接口。
- 命令行位置参数由“必须同 Host”改为一个或多个入口；默认 Scope 仍为精确入口 Host 集合。
- 不修改 Tanscan 对外 API、数据库结构、任务记录模型或 CrawlerGo SSRF 行为。
- 更新 CrawlerGo 用法说明，并重建 Tanscan 随附的 Windows/Linux CrawlerGo 二进制。

## 验证

- CrawlerGo 单测覆盖三种 Scope、多入口、多注册根域、越界 Host、Scheme 限制、唯一入口扩展去重与子域统计。
- Tanscan 单测验证一次运行收到全部 URL 和 Scope 参数，并只读取一个聚合输出文件。
- 两个仓库分别运行 Go 测试；更新 Windows amd64 和 Linux amd64 二进制并检查构建结果。
