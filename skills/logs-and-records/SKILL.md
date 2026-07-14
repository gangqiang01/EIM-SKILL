---
name: logs-and-records
description: Turn a device's system log collection on/off, query and delete DeviceOn-EIM admin operation records (audit log), and request/download a device agent's client log. Use this whenever the user asks about audit trail/operation history or wants to fetch an agent's client log via the eim-skill CLI. 中文触发词：操作记录、操作日志、审计日志、系统日志采集、Agent日志、客户端日志、下载日志。
---

# Logs and operation records

Wraps two related but distinct log/audit sources on the main AppHub
server, plus a device-side system-log-collection toggle. Requires
`eim-skill` built and configured (see repo README).

## When to use this skill

- "Turn on/off system log collection for device X" / "开启/关闭设备X的系统日志收集" → `log system set-monitor`
- "Who did what, and when?" (audit trail) / "查看操作记录" / "查看操作日志/审计日志" → `log operations list`
- "Clear the audit log" / "清空操作记录" → `log operations delete` / `delete-all` / `batch-delete`
- "Get device X's agent/client log" / "获取设备X的Agent日志" / "下载设备X的客户端日志" → `log agent request` then `log agent status` then `log agent download`

## Commands

```
eim-skill log system set-monitor <endpoint-or-name> <on|off>

eim-skill log operations list [--limit N] [--target <substr>] [--last <30m|12h|3d>]
eim-skill log operations delete <id>
eim-skill log operations delete-all
eim-skill log operations batch-delete <id> [id...]

eim-skill log agent request <endpoint-or-name> <callback-url> <action-id>
eim-skill log agent status <action-id>
eim-skill log agent download <endpoint-or-name> [--out <file>]
```

`<endpoint-or-name>` accepts either the device's endpoint id or its
human-readable device name (`devName`) - resolved against the live device
list.

## Notes and caveats

- **Agent log retrieval is asynchronous and requires a reachable
  callback server.** The AppHub server can't push a device's log file to
  you directly - `log agent request` asks the device's agent to upload its
  log to `<callback-url>`, which must be an HTTP address the device can
  actually reach, running a server that accepts the agent's upload. This
  CLI does not provide that callback server. Workflow: `request` → poll
  `status` until it reports `"success"` → `download`. The download can
  only happen once; the server deletes the file after serving it.
- `log operations list` is not documented in AppHub-v2.3.0.md - it was
  reverse-engineered from `OperationRecordServlet.java` and its field
  names/shapes (`orid`, `type`, `event`, `target`, `username`, `ts`) are
  confirmed against a live server's actual response (`data` is a plain
  JSON array, not the double-encoded string the servlet name would
  suggest).
- `--last` (on `log operations list`) takes a relative window (e.g. `30m`,
  `12h`, `3d`) covering that far back through now, instead of raw
  Unix-millisecond timestamps; omit it for no time filter. `--limit`
  (default 20) caps how many of the most recent entries come back - there's
  no `--page` to think about.
- `log system set-monitor` is not documented in AppHub-v2.3.0.md either
  (inferred from a captured devicecontrol request using the
  `set_monitor_data_report`/`systemd_log_monitor` action), but is confirmed
  to work against a live device (its effect shows up afterward in
  `log operations list`'s audit trail). There is currently **no working
  command to query the system log entries it enables collection of** - the
  documented `/api/systemLog/bypage` endpoint returned an HTTP 400
  "Invalid path" error from a real AppHub server during testing (a
  server-side routing problem, not a request-shape problem - the request
  matched AppHub-v2.3.0.md exactly), so the `list`/`delete`/
  `delete-by-device`/`batch-delete` commands that depended on it were
  removed rather than shipped non-functional. If a future AppHub
  environment fixes that routing issue, those commands would need to be
  re-added.
- Every command prints one JSON object to stdout:
  `{"status":200,"msg":"","data":...}` or `{"status":500,"msg":"..."}`, with
  exit code 0/1 to match. Don't confuse that outer `status` (did the CLI
  call itself succeed - 200/500) with `log agent status`'s inner
  `data.result` field (`"running"`/`"success"`/`"timeout"`/`"error"` -
  whether the *device's* log upload finished).
