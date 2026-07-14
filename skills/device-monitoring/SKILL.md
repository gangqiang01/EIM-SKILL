---
name: device-monitoring
description: Read and configure a DeviceOn-EIM device's monitoring and alerting - a live CPU/memory/storage/battery/network snapshot, CPU/memory/storage/network/battery thresholds, application/process monitoring, Docker container monitoring, USB peripheral change monitoring, and historical CPU/memory samples. Use this whenever the user asks about a device's resource usage, alert thresholds, or wants to turn monitoring of some subsystem on/off via the eim-skill CLI. 中文触发词：设备监控、监控详情、CPU监控、内存监控、存储监控、网络监控、告警阈值、阈值设置、进程监控、应用监控、Docker监控、容器监控、USB外设监控、U盘监控。
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
- "Turn hardware monitoring on/off for device X" / "开启/关闭设备X的硬件监控" → `device set-monitor`
- "What processes are running on device X, and is process monitoring on?" / "查看设备X的进程列表" / "开启/关闭进程监控" → `device process list` / `device process set-monitor`
- "Which processes are currently being monitored on device X?" / "查看设备X正在监控的进程" → `device process monitored`
- "What Docker containers are on device X?" / "查看设备X的Docker容器" → `device docker info`
- "Turn on USB-change alerts for device X" / "开启设备X的USB外设变更告警" → `device usb set-monitor`
- "Allow/deny USB storage access on device X" / "允许/禁止设备X的U盘存储访问" → `device usb set-storage` (unverified, see caveat below 未验证，见下方说明)
- "Show me CPU/memory history for device X" / "查看设备X的CPU/内存历史监控数据" → `device monitor --type cpu|memory`

## Commands

```
eim-skill device get-monitor <endpoint-or-name>                          # live cpu/memory/storage/battery/network snapshot

eim-skill device set-monitor <endpoint-or-name> <on|off>                # master hw-monitoring toggle

eim-skill device threshold get <endpoint-or-name>
eim-skill device threshold set <endpoint-or-name> <metric> <on|off> [--warn N] [--error N]
  # metric: cpu_temp, cpu_usage, memory_usage, storage_usage,
  #         battery_available, network_upload_speed, network_download_speed

eim-skill device process list <endpoint-or-name>
eim-skill device process monitored <endpoint-or-name>
eim-skill device process set-monitor <endpoint-or-name> <on|off>

eim-skill device docker info <endpoint-or-name>
eim-skill device docker set-monitor <endpoint-or-name> <on|off>

eim-skill device usb get-monitor <endpoint-or-name>                      # USB change-alerting, distinct from
eim-skill device usb set-monitor <endpoint-or-name> <on|off>             # usb set-storage (mass-storage enable/disable)
eim-skill device usb set-storage <endpoint-or-name> <on|off>             # mass-storage access, unverified (see below)

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
  real numbers, `is_monitored` a real boolean. Its per-item shape is
  inferred by analogy with `device process list` (same subsystem), not
  verified against a populated list.
- CPU/memory/storage/network/battery are monitored via configurable
  warn/error *thresholds*, not a live value feed. Pair `device threshold
  get` (current config) with `device monitor --type cpu|memory`
  (historical samples) - storage/network have no persisted history
  server-side.
- `device monitor --last` takes a relative window (e.g. `30m`, `12h`,
  `3d`) covering that far back through now, instead of raw Unix-millisecond
  timestamps - omit it to get all available samples.
- Process and Docker monitoring here are on/off toggles only; the web
  console's per-process/per-container selection (pick exactly which
  processes/containers to watch) isn't wired up in this CLI.
- `device usb get-monitor`/`set-monitor` (USB *change* alerting - alert on
  plug/unplug) is a different feature from `device usb set-storage`
  (whether USB mass storage itself is allowed).
- `device usb set-storage` is unverified: not documented in
  AppHub-v2.3.0.md, and its sibling read command (`usb get-storage`) was
  removed after live testing showed the server didn't recognize the
  guessed action name it queried (it silently returned an empty result
  instead of a real flag). The write-side action name may be equally
  wrong - verify it actually changes device behavior before relying on it.
- Every command prints one JSON object to stdout:
  `{"status":200,"msg":"","data":...}` or `{"status":500,"msg":"..."}`, with
  exit code 0/1 to match.
