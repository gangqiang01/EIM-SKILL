---
name: device-monitoring
description: Read and configure a DeviceOn-EIM device's monitoring and alerting - a live CPU/memory/storage/battery/network snapshot, CPU/memory/storage/network/battery thresholds, application/process monitoring, Docker container monitoring, USB peripheral change monitoring, and historical CPU/memory samples. Use this whenever the user asks about a device's resource usage, alert thresholds, or wants to turn monitoring of some subsystem on/off via the eim-skill CLI. 中文触发词：设备监控、监控详情、CPU监控、内存监控、存储监控、网络监控、告警阈值、阈值设置、进程监控、应用监控、Docker监控、容器监控、USB外设监控。
---

# Device monitoring and alerting

Wraps DeviceOn-EIM's per-device monitoring config, which is
threshold/alert based (not a live streaming dashboard) - this mirrors the
web console's "monitorAlert" feature. Requires `eim-skill` built and
configured (see repo README).

## Running the eim-skill binary / 如何运行 eim-skill 二进制

Commands below are written as `../scripts/eim-skill ...`, assuming a
Linux x86_64 host and this file's own directory as the working
directory - the binary ships in a `scripts/` folder that sits alongside
every skill folder (i.e. one level up from this file, at
`../scripts/`), not on PATH. On a different host, substitute the
matching binary name everywhere `../scripts/eim-skill` appears below:

- Linux arm64/aarch64: `../scripts/eim-skill-arm64`
- Windows: `../scripts/eim-skill.exe`

下面的命令统一写成 `../scripts/eim-skill ...`，默认假设运行在 Linux
x86_64 主机上，且执行时的工作目录就是这个文件所在的目录——二进制文件
放在和每个技能目录同级的 `scripts/` 目录下（即这个文件所在目录的
上一级，路径为 `../scripts/`），而不在 PATH 里。如果运行在其他平台，
请把下面所有出现的 `../scripts/eim-skill` 替换成对应的二进制名：

- Linux arm64/aarch64：`../scripts/eim-skill-arm64`
- Windows：`../scripts/eim-skill.exe`

## When to use this skill

- "What's device X's current CPU/memory/storage/battery/network usage?" / "查看设备X的监控详情" → `device get-monitor`
- "Is CPU/memory/storage/network monitoring on for device X, and what are
  the alert thresholds?" / "查看设备X的监控阈值" → `device threshold get`
- "Alert me if CPU usage on device X goes above 90%" / "设置设备X的CPU使用率告警阈值" → `device threshold set`
- "What processes are running on device X?" / "查看设备X的进程列表" → `device process list`
- "Which processes are currently being monitored on device X?" / "查看设备X正在监控的进程" → `device process monitored`
- "Monitor/stop monitoring process (pid) N on device X" / "监控/取消监控设备X的N号进程" → `device process set-monitor <endpoint> <pid> on|off`
- "What Docker containers are on device X?" / "查看设备X的Docker容器" → `device docker list`
- "Which Docker containers on device X are being monitored?" / "查看设备X正在监控的Docker容器" → `device docker monitored`
- "Monitor/stop monitoring container X on device Y" / "监控/取消监控设备Y上的X容器" → `device docker set-monitor <endpoint> <container-name> on|off`
- "Turn on USB-change alerts for device X" / "开启设备X的USB外设变更告警" → `device usb set-monitor`
- "Allow/block USB storage devices (like a flash drive) on device X" / "允许/禁止设备X使用USB存储设备(U盘)" → `device usb set-storage`
- "Is device X allowed to use USB storage devices right now?" / "设备X现在能不能用USB存储设备" → `device usb get-storage`
- "Show me CPU/memory history for device X" / "查看设备X的CPU/内存历史监控数据" → `device monitor --type cpu|memory`
- "Is CPU/memory historical-data collection on for device X?" / "设备X的CPU/内存历史数据采集开着吗" → `device get-monitor-report`
- "Turn on/off CPU/memory data collection for device X" / "开启/关闭设备X的CPU/内存数据采集" → `device set-monitor-report`

## Commands

```
../scripts/eim-skill device get-monitor <endpoint-or-name>                          # live cpu/memory/storage/battery/network snapshot

../scripts/eim-skill device threshold get <endpoint-or-name>
../scripts/eim-skill device threshold set <endpoint-or-name> <metric> <on|off> [--warn N] [--error N]
  # metric: cpu_temp, cpu_usage, memory_usage, storage_usage,
  #         battery_available, network_upload_speed, network_download_speed

../scripts/eim-skill device process list <endpoint-or-name>
../scripts/eim-skill device process monitored <endpoint-or-name>
../scripts/eim-skill device process set-monitor <endpoint-or-name> <pid> <on|off>

../scripts/eim-skill device docker list <endpoint-or-name>
../scripts/eim-skill device docker monitored <endpoint-or-name>
../scripts/eim-skill device docker set-monitor <endpoint-or-name> <container-name> <on|off>

../scripts/eim-skill device usb get-monitor <endpoint-or-name>                      # is USB change-alerting on?
../scripts/eim-skill device usb set-monitor <endpoint-or-name> <on|off>             # USB change-alerting on/off
../scripts/eim-skill device usb get-storage <endpoint-or-name>                      # is USB mass-storage device access enabled?
../scripts/eim-skill device usb set-storage <endpoint-or-name> <on|off>             # USB mass-storage device access on/off

../scripts/eim-skill device monitor <endpoint-or-name> --type cpu|memory [--last <30m|12h|3d>]
  # or an absolute range instead of --last: [--begin <time>] [--end <time>]
../scripts/eim-skill device get-monitor-report <endpoint-or-name>
../scripts/eim-skill device set-monitor-report <endpoint-or-name> <cpu|memory> <on|off>
```

`<endpoint-or-name>` accepts either the device's endpoint id or its
human-readable device name (`devName`) - resolved against the live device
list, same as in the device-management skill.

## Notes and caveats

- `device get-monitor` is undocumented in AppHub-v2.3.0.md and its output is
  normalized: the raw agent payload mixes real numbers, numeric strings,
  and `"N/A"` placeholders, so this CLI parses everything into consistent
  `float64` fields (`null` when unavailable). `*_percent`/`*_gb` units are
  inferred from observed values (e.g. memory's free+usage and storage's
  free+usage each summed to 100), not from a spec.
- `device process monitored` is also undocumented (the `get_monitor_app`
  action) and normalized the same way - `cpu_percent`/`mem_percent` and
  `pid` are real numbers (the agent reports `pid` as a numeric string,
  e.g. `"pid":"639"`). `is_monitored` is always `true` for every item this
  returns, not conditionally decoded from the agent's own
  `threadismonitor` field as an earlier version assumed - confirmed
  against a live server that `get_monitor_app` never actually populates
  that field (always empty), so being present in this list at all is
  what "monitored" means for this action.
- `device process list`/`device docker list` return every process/Docker
  container regardless of monitor status; `device process monitored`/
  `device docker monitored` filter to just the monitored subset
  client-side - same underlying action, no extra request. `device docker
  list`'s endpoint is confirmed against a live server via a real packet
  capture; an earlier version instead read an undocumented LwM2M
  resource (`.../60001/0/27300`) assuming a response grouped by compose
  project, which returned an
  empty result on a live server with real
  containers deployed - fixed via the packet capture.
- `device process set-monitor`/`device docker set-monitor` add/remove one
  process (by pid) or container (by name) to/from the monitored list -
  you only give the identifier; the underlying API actually requires
  resending the *entire* process/container object (from `list`) with an
  add/remove flag, so this fetches the current list itself, finds the
  matching item, and resends it verbatim. Confirmed against a live server
  via a real packet capture and cross-verified end-to-end (adding makes
  it appear in `monitored`, removing makes it disappear, without
  disturbing anything else already monitored). No single-device version
  of the process one is documented in AppHub-v2.3.0.md - only a
  batch/group endpoint is (`batchsetsoftwaremonitor`), which reliably
  fails with an empty-bodied HTTP 500 on a live server (matching another
  known-broken batch endpoint), so the undocumented single-device path is
  used instead.
- `device usb get-monitor`/`set-monitor` control USB peripheral
  change-alerting (alert on connect/disconnect), using `get_usb_change`/
  `set_peripheral_monitor` - confirmed against a live server via a real
  packet capture and cross-verified by toggling `set-monitor` true/false
  and observing `get-monitor` correctly reflects each change. (A brief
  earlier version swapped in `get_usb_storage_status`/
  `set_usb_storage_status` instead - a different, separate feature, USB
  mass-storage access, based on a packet capture that turned out to be
  for that other feature. Its SET call succeeded and was logged under a
  friendly audit-log event name, which looked like good evidence, but its
  GET side never reflected the change on a live device tested, so it was
  reverted - a GET that actually reflects the SET is stronger evidence of
  a real toggle than an audit-log label.)
- `device usb get-storage`/`set-storage` are a separate, independent
  switch from `get-monitor`/`set-monitor` - whether the device allows USB
  mass-storage devices (e.g. a USB flash drive) to be used, not
  connect/disconnect alerting. `set-storage` is confirmed via
  `set_usb_storage_status`, cross-checked in `log operations list`'s
  audit trail (`on` logged as `enableUsb`, `off` as `disableUsb`);
  `get-storage` uses `get_usb_storage_status` and was cross-verified on a
  live Windows device by toggling `set-storage` and observing it
  correctly reflects each change in real time. Not every device supports
  this action - a Linux industrial PC tested returned an empty result
  regardless of the last value set, normalized to `enabled: false` here.
- CPU/memory/storage/network/battery are monitored via configurable
  warn/error *thresholds*, not a live value feed. Pair `device threshold
  get` (current config) with `device monitor --type cpu|memory`
  (historical samples) - storage/network have no persisted history
  server-side.
- `device threshold set` only overwrites `--warn`/`--error` when they're
  explicitly passed. If either is omitted (e.g. just toggling a metric
  on/off), it fetches the metric's current warn/error levels first (via
  `device threshold get`) and resends those unchanged - the underlying
  `set_threshold` action requires both on every request, and an earlier
  version sent `0` for an omitted flag, silently zeroing out an existing
  threshold on every on/off toggle (confirmed via a real bug report and
  packet capture).
- `device threshold set` validates `--warn`/`--error` client-side, since
  the server itself accepts anything - confirmed live that it silently
  stores `warn=150`/`error=200`, or even negative values, verbatim for
  `cpu_usage`. Percentage metrics (`cpu_usage`, `memory_usage`,
  `storage_usage`, `battery_available`) must be within 0-100, neither may
  be negative, and the pair must be ordered correctly for the metric's
  direction: `warn < error` for every metric except `battery_available`
  (lower is worse there, so it's reversed: `warn > error`, e.g. `warn=20,
  error=10`). These checks only apply to values you actually pass - a pure
  on/off toggle with no `--warn`/`--error` skips them.
- `device monitor --last` takes a relative window (e.g. `30m`, `12h`,
  `3d`) covering that far back through now, instead of raw Unix-millisecond
  timestamps - omit it to get all available samples. `--begin`/`--end` are
  an alternative, absolute way to specify the range (RFC3339 or
  `"YYYY-MM-DD HH:MM"`, e.g. `"2026-07-01 00:00"`) - either or both may be
  given (an omitted bound is unbounded), and they're mutually exclusive
  with `--last`.
- `device monitor`'s per-sample `value` is parsed leniently (JSON number
  or numeric string, both normalized to a plain number) - some servers
  report it as a string, which an earlier version failed to decode
  (confirmed and fixed via a real bug report).
- `device get-monitor-report`/`set-monitor-report` control whether CPU/
  memory historical-data collection is happening at all - if a type is
  off, `device monitor --type cpu|memory` has nothing new to show for
  it. Uses `set_monitor_data_report`/`get_monitor_data_report`
  (confirmed via a real packet capture and live write/readback testing)
  - the same action `log system set-monitor`/`get-monitor` uses for
  system logs, just a different key (`cpu_monitor`/`memory_monitor` vs
  `systemd_log_monitor`) in the same response.
- There is no `device set-monitor` (overall hardware monitoring on/off) -
  it used the `set_isMonitor` action and was removed; `device threshold
  set` already lets you disable alerting per-metric. There is also no
  `device process set-monitor`/`device docker set-monitor`: an earlier
  version had on/off toggles for these (`set_isProcessMonitor`/
  `set_isDockerMonitor`), but they were confirmed against a live server
  to return success without actually changing anything, so they were
  removed.
- Every command prints one JSON object to stdout:
  `{"status":200,"msg":"","data":...}` or `{"status":500,"msg":"..."}`, with
  exit code 0/1 to match.
