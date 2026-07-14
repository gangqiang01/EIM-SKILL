---
name: logs-and-records
description: Query and delete DeviceLink/AppHub device system logs, admin operation records (audit log), and request/download a device agent's client log. Use this whenever the user asks to review device logs, audit trail/operation history, or fetch an agent's client log via the eim-skill CLI. 中文触发词：设备日志、系统日志、操作记录、操作日志、审计日志、Agent日志、客户端日志、下载日志。
---

# Logs and operation records

Wraps three related but distinct log sources on the main AppHub server.
Requires `eim-skill` built and configured (see repo README).

## When to use this skill

- "Show me device X's system log" / "查看设备X的系统日志" → `log system list`
- "Clear/delete some system log entries" / "删除系统日志" → `log system delete` / `delete-by-device` / `batch-delete`
- "Who did what, and when?" (audit trail) / "查看操作记录" / "查看操作日志/审计日志" → `log operations list`
- "Clear the audit log" / "清空操作记录" → `log operations delete` / `delete-all` / `batch-delete`
- "Get device X's agent/client log" / "获取设备X的Agent日志" / "下载设备X的客户端日志" → `log agent request` then `log agent status` then `log agent download`

## Commands

```
eim-skill log system list [--page N] [--limit N] [--device-id <endpoint-or-name>] [--os <os>]
  [--keywords <kw>] [--begin <unix-ms>] [--end <unix-ms>]
eim-skill log system delete <id>
eim-skill log system delete-by-device <endpoint-or-name>
eim-skill log system batch-delete <id> [id...]

eim-skill log operations list [--page N] [--limit N] [--target <substr>]
  [--begin <unix-ms>] [--end <unix-ms>]
eim-skill log operations delete <id>
eim-skill log operations delete-all
eim-skill log operations batch-delete <id> [id...]

eim-skill log agent request <endpoint-or-name> <callback-url> <action-id>
eim-skill log agent status <action-id>
eim-skill log agent download <endpoint-or-name> [--out <file>]
```

`<endpoint-or-name>` (and `--device-id`) accept either the device's
endpoint id or its human-readable device name (`devName`) - resolved
against the live device list.

## Notes and caveats

- **Agent log retrieval is asynchronous and requires a reachable
  callback server.** The AppHub server can't push a device's log file to
  you directly - `log agent request` asks the device's agent to upload its
  log to `<callback-url>`, which must be an HTTP address the device can
  actually reach, running a server that accepts the agent's upload. This
  CLI does not provide that callback server. Workflow: `request` → poll
  `status` until it reports `"success"` → `download`. The download can
  only happen once; the server deletes the file after serving it.
- `log operations list` is not documented in AppHub-v2.3.0.md - it's a
  code-only feature (see `OperationRecordServlet.java`).
- Every command prints one JSON object to stdout:
  `{"status":200,"msg":"","data":...}` or `{"status":500,"msg":"..."}`, with
  exit code 0/1 to match. Don't confuse that outer `status` (did the CLI
  call itself succeed - 200/500) with `log agent status`'s inner
  `data.result` field (`"running"`/`"success"`/`"timeout"`/`"error"` -
  whether the *device's* log upload finished).
