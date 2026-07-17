---
name: logs-and-records
description: Turn a device's system log collection on/off, query/delete its collected system log entries, and query/delete DeviceOn-EIM admin operation records (audit log). Use this whenever the user asks about a device's system log, or about audit trail/operation history, via the eim-skill CLI. 中文触发词：操作记录、操作日志、审计日志、系统日志采集、系统日志查询、设备日志。
---

# Logs and operation records

Wraps two related but distinct log/audit sources on the main DeviceOn-EIM
server: a per-device system log (collection toggle + query/delete), and
the server's admin operation records (audit log). Requires `eim-skill`
built and configured (see repo README).

## When to use this skill

- "Turn on/off system log collection for device X" / "开启/关闭设备X的系统日志收集" → `log system set-monitor`
- "Show me device X's recent system log" / "查看设备X的系统日志" → `log system list --device-id X`
- "Clear device X's system log" / "清空设备X的系统日志" → `log system delete-by-device`
- "Who did what, and when?" (audit trail) / "查看操作记录" / "查看操作日志/审计日志" → `log operations list`
- "Clear the audit log" / "清空操作记录" → `log operations delete-all`

## Commands

```
eim-skill log system set-monitor <endpoint-or-name> <on|off>

eim-skill log system list [--device-id <endpoint-or-name>] [--os <os>] [--keywords <substr>]
                          [--mode <mode>] [--level <level>] [--limit N] [--last <30m|12h|3d>]
eim-skill log system delete-by-device <endpoint-or-name>

eim-skill log operations list [--limit N] [--target <substr>] [--last <30m|12h|3d>]
eim-skill log operations delete-all
```

`<endpoint-or-name>` accepts either the device's endpoint id or its
human-readable device name (`devName`) - resolved against the live device
list.

## Notes and caveats

- `log system list` requires `log system set-monitor <device> on` to have
  been run against a device at least once - otherwise there's nothing to
  report for it. There is no single-id delete or batch-delete for system
  log entries - `delete-by-device` (clear everything for one device) is
  the only removal command.
- `log system set-monitor` and `log system list`/`delete-by-device` are
  not documented in AppHub-v2.3.0.md - reverse-engineered from a captured
  devicecontrol request (`set_monitor_data_report`/`systemd_log_monitor`)
  and a real packet capture of the query endpoint, respectively; both are
  confirmed against a live server. The query endpoint's path is
  `/api/systemLog/byPage` (capital `P`) - an earlier version of this CLI
  used the lowercase `/api/systemLog/bypage`, which the server rejects
  with an HTTP 400 "Invalid path" error, and this whole command group had
  been removed as a result before that bug (not a genuine server
  limitation) was found and fixed.
- `log operations list` is not documented in AppHub-v2.3.0.md - it was
  reverse-engineered from `OperationRecordServlet.java` and its field
  names/shapes (`orid`, `type`, `event`, `target`, `username`, `ts`) are
  confirmed against a live server's actual response (`data` is a plain
  JSON array, not the double-encoded string the servlet name would
  suggest).
- `--last` (on `log system list` and `log operations list`) takes a
  relative window (e.g. `30m`, `12h`, `3d`) covering that far back through
  now, instead of raw Unix-millisecond timestamps; omit it for no time
  filter. `--limit` (default 20) caps how many of the most recent entries
  come back - there's no `--page` to think about.
- Every command prints one JSON object to stdout:
  `{"status":200,"msg":"","data":...}` or `{"status":500,"msg":"..."}`, with
  exit code 0/1 to match.
