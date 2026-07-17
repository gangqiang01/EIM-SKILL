---
name: device-monitoring
description: Read and configure a DeviceOn-EIM device's monitoring and alerting - a live CPU/memory/storage/battery/network snapshot, CPU/memory/storage/network/battery thresholds, application/process monitoring, Docker container monitoring, USB peripheral change monitoring, and historical CPU/memory samples. Use this whenever the user asks about a device's resource usage, alert thresholds, or wants to turn monitoring of some subsystem on/off via the eim-skill CLI. 中文触发词：设备监控、监控详情、CPU监控、内存监控、存储监控、网络监控、告警阈值、阈值设置、进程监控、应用监控、Docker监控、容器监控、USB外设监控。
---

# Device monitoring and alerting

Wraps DeviceOn-EIM's per-device monitoring config, which is
threshold/alert based (not a live streaming dashboard) - this mirrors the
web console's "monitorAlert" feature. Requires `eim-skill` built and
configured (see repo README).

## When to use this skill

- "What's device X's current CPU/memory/storage/battery/network usage?" / "查看设备X的监控详情" → `device get-monitor`
- "Is CPU/memory/storage/network monitoring on for device X, and what are
  the alert thresholds?" / "查看设备X的监控阈值" → `device threshold get`
- "Alert me if CPU usage on device X goes above 90%" / "设置设备X的CPU使用率告警阈值" → `device threshold set`
- "Which processes are currently being monitored on device X?" / "查看设备X正在监控的进程" → `device process monitored`
- "Which Docker containers on device X are being monitored?" / "查看设备X正在监控的Docker容器" → `device docker monitored`
- "Turn on USB-change alerts for device X" / "开启设备X的USB外设变更告警" → `device usb set-monitor`
- "Show me CPU/memory history for device X" / "查看设备X的CPU/内存历史监控数据" → `device monitor --type cpu|memory`

## Commands

```
eim-skill device get-monitor <endpoint-or-name>                          # live cpu/memory/storage/battery/network snapshot

eim-skill device threshold get <endpoint-or-name>
eim-skill device threshold set <endpoint-or-name> <metric> <on|off> [--warn N] [--error N]
  # metric: cpu_temp, cpu_usage, memory_usage, storage_usage,
  #         battery_available, network_upload_speed, network_download_speed

eim-skill device process monitored <endpoint-or-name>

eim-skill device docker monitored <endpoint-or-name>

eim-skill device usb get-monitor <endpoint-or-name>                      # is USB change-alerting on?
eim-skill device usb set-monitor <endpoint-or-name> <on|off>             # USB change-alerting on/off

eim-skill device monitor <endpoint-or-name> --type cpu|memory [--last <30m|12h|3d>]
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
  action) and normalized the same way - `cpu_percent`/`mem_percent` are
  real numbers, `is_monitored` a real boolean, `pid` a real number (the
  agent reports it as a numeric string, e.g. `"pid":"639"` - confirmed
  against a live server). Its per-item shape is inferred by analogy with
  the sibling `get_software_info` action (same subsystem), not verified
  against a populated list.
- `device docker monitored` returns only the Docker containers currently
  selected for monitoring (`ismonitor == true`), not every deployed
  container - it's the Docker analogue of `device process monitored`.
  Uses the undocumented `get_docker_info` devicecontrol action (confirmed
  against a live server via a real packet capture), which itself returns
  every container regardless of monitor status; this CLI filters to the
  monitored subset. An earlier version instead read an undocumented LwM2M
  resource (`.../60001/0/27300`) assuming a response grouped by compose
  project, which returned an empty result on a live server with real
  containers deployed - fixed via the packet capture.
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
  a real toggle than an audit-log label.) USB mass-storage access itself
  isn't currently exposed as a command, since its GET side couldn't be
  confirmed working.
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
- `device monitor --last` takes a relative window (e.g. `30m`, `12h`,
  `3d`) covering that far back through now, instead of raw Unix-millisecond
  timestamps - omit it to get all available samples.
- `device monitor`'s per-sample `value` is parsed leniently (JSON number
  or numeric string, both normalized to a plain number) - some servers
  report it as a string, which an earlier version failed to decode
  (confirmed and fixed via a real bug report).
- There is no `device set-monitor` (overall hardware monitoring on/off) -
  it used the `set_isMonitor` action and was removed; `device threshold
  set` already lets you disable alerting per-metric. There is also no
  `device process set-monitor`/`device docker set-monitor`: an earlier
  version had on/off toggles for these (`set_isProcessMonitor`/
  `set_isDockerMonitor`), but they were confirmed against a live server
  to return success without actually changing anything, so they were
  removed.
- There is no `device process list` (every process regardless of monitor
  status) - only `device process monitored` remains.
- Every command prints one JSON object to stdout:
  `{"status":200,"msg":"","data":...}` or `{"status":500,"msg":"..."}`, with
  exit code 0/1 to match.
