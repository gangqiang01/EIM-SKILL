---
name: logs-and-records
description: Turn a device's system log collection on/off, query/delete its collected system log entries, download a device's agent (client-side) log or one specific file from its filesystem by absolute path, and query/delete DeviceOn-EIM admin operation records (audit log). Use this whenever the user asks about a device's system log, agent log, a specific file/log path on a device, or about audit trail/operation history, via the eim-skill CLI. 中文触发词：操作记录、操作日志、审计日志、系统日志采集、系统日志查询、设备日志、agent日志、客户端日志、获取设备文件、指定路径文件。
---

# Logs and operation records

Wraps three related but distinct log/audit sources on the main
DeviceOn-EIM server: a per-device system log (collection toggle +
query/delete), a per-device agent (client-side) log (download only), and
the server's admin operation records (audit log). Requires `eim-skill`
built and configured (see repo README).

## Running the eim-skill binary / 如何运行 eim-skill 二进制

Commands below are written as `../../eim-skill ...`, assuming a
Linux x86_64 host and this file's own directory as the working
directory - the binary ships at the package root, two levels up from
this file (`../../`), alongside `conf/`; there is no separate copy under
a `scripts/` folder. On a different host, substitute the matching
binary name everywhere `../../eim-skill` appears below:

- Linux arm64/aarch64: `../../eim-skill-arm64`
- Windows: `../../eim-skill.exe`

下面的命令统一写成 `../../eim-skill ...`，默认假设运行在 Linux
x86_64 主机上，且执行时的工作目录就是这个文件所在的目录——二进制文件
放在包根目录下（即这个文件所在目录的上两级，路径为 `../../`），和
`conf/` 同级，不再单独放一份 `scripts/` 目录的副本。如果运行在其他
平台，请把下面所有出现的 `../../eim-skill` 替换成对应的二进制名：

- Linux arm64/aarch64：`../../eim-skill-arm64`
- Windows：`../../eim-skill.exe`

## When to use this skill

- "Turn on/off system log collection for device X" / "开启/关闭设备X的系统日志收集" → `log system set-monitor`
- "Is system log collection on for device X?" / "设备X的系统日志收集开着吗" → `log system get-monitor`
- "Show me device X's recent system log" / "查看设备X的系统日志" → `log system list --device-id X`
- "Clear device X's system log" / "清空设备X的系统日志" → `log system delete-by-device`
- "Download device X's agent/client log" / "下载设备X的agent日志/客户端日志" → `log agent download`
- "Get file <path> from device X" / "获取设备X上 <路径> 这个文件" → `log agent file`
- "Who did what, and when?" (audit trail) / "查看操作记录" / "查看操作日志/审计日志" → `log operations list`
- "Clear the audit log" / "清空操作记录" → `log operations delete-all`

## Commands

```
../../eim-skill log system set-monitor <endpoint-or-name> <on|off>
../../eim-skill log system get-monitor <endpoint-or-name>

../../eim-skill log system list [--device-id <endpoint-or-name>] [--os <os>] [--keywords <substr>]
                          [--mode <mode>] [--level <level>] [--limit N] [--last <30m|12h|3d>]
                          # or an absolute range instead of --last: [--begin <time>] [--end <time>]
../../eim-skill log system delete-by-device <endpoint-or-name>

../../eim-skill log agent download <endpoint-or-name> [--output <path>]
                             [--content [--lines N]]  # return the latest day's log tail directly instead of just saving a file

../../eim-skill log agent file <endpoint-or-name> <remote-path> [--lines N] [--output <path>]

../../eim-skill log operations list [--limit N] [--target <substr>] [--last <30m|12h|3d>]
                              # or an absolute range instead of --last: [--begin <time>] [--end <time>]
../../eim-skill log operations delete-all
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
  limitation) was found and fixed. `level` and `pid` are normalized to
  real JSON numbers - the server reports both as numeric strings (e.g.
  `"level":"4"`), confirmed against a live server.
- `log system get-monitor` reports whether system log collection is
  currently on, via `get_monitor_data_report` - the same devicecontrol
  action `device get-monitor-report` uses for CPU/memory collection
  status (a shared skill in `device-monitoring`), just reading a
  different key (`systemd_log_monitor`) from the same response.
  Confirmed against a live server.
- `log agent download` is a three-step, undocumented (in
  AppHub-v2.3.0.md) process confirmed against a live server via a real
  packet capture: trigger a push (LwM2M write to resource 40001/0/27012,
  with a fresh actionId and this server's own address), poll
  `/api/devicemgr/logStatus?actionId=...` every 2 seconds up to a 3-minute
  timeout until it reports "success", then fetch and save
  `/api/devicemgr/agentLog/{endpoint}`. The downloaded file is a zip
  archive of per-day log files plus the current day's log - the endpoint
  sends no `Content-Type` header, confirmed live, but the body is
  reliably a zip by its magic bytes. This is a different log from `log
  system` (that's the device's systemd/OS log, forwarded to and queried
  from the server) - this is the DeviceOn-EIM client process's own log.
- `log agent download --content` extracts the zip and returns the tail of
  the most recent day's log entry directly as text (`data.content`), for
  when the caller just wants to look at recent log lines rather than
  handle a file - the archive's smallest daily file observed live was
  still several MB, so returning the whole thing isn't practical.
  "Most recent" prefers an entry named exactly `EIM-Client.log` (the
  currently-active log, confirmed live to duplicate that day's dated
  `YYYYMMDD.log` entry) when present. `--lines` (default 200) controls how
  many trailing lines come back. Without `--output` given alongside
  `--content`, the zip is not saved to disk at all - pass `--output` too
  if you want both.
- `log agent file <endpoint-or-name> <remote-path>` fetches one specific
  file from a device's filesystem by absolute path (e.g.
  `/media/recovery/EIM-Client/20260721.log`) - not documented in
  AppHub-v2.3.0.md, confirmed against a live server via a real packet
  capture. Shares `log agent download`'s trigger/poll/fetch shape, but
  against a different LwM2M resource (40001/0/27023, which takes the file
  path directly) and a different download endpoint
  (`/api/devicemgr/log?filename=<endpoint>_<label>.zip&type=app&clientId=<endpoint>&value=<label>`,
  not `/api/devicemgr/agentLog`) that requires a label ("appname" in the
  trigger, "value" in the download query) tying the two together -
  confirmed against a live server that this reuses `actionId` for that
  label, since nothing in the capture suggests it needs to mean anything
  else. Unlike the agent log dump, the returned zip always contains
  exactly one file (confirmed live - fetching a nonexistent path fails the
  trigger step outright with a clear server error, "specificFiles is
  empty", rather than returning an empty/wrong file), so its content is
  printed directly by default (as text; `--output` saves the raw file
  instead, e.g. for binary content or to keep a copy). `--lines` returns
  just the trailing N lines instead of the whole file.
- `log operations list` is not documented in AppHub-v2.3.0.md - it was
  reverse-engineered from `OperationRecordServlet.java` and its field
  names/shapes (`orid`, `type`, `event`, `target`, `username`, `ts`) are
  confirmed against a live server's actual response (`data` is a plain
  JSON array, not the double-encoded string the servlet name would
  suggest). `event` is normalized (`FlexibleJSON`, not a plain string) -
  the server encodes it inconsistently across entries (a raw JSON object
  for some, a JSON string containing escaped JSON for others, a bare
  number or plain string for others still), confirmed via a real decode
  error hit live ("cannot unmarshal object into Go struct field
  OperationRecord.data.event of type string").
- `--last` (on `log system list` and `log operations list`) takes a
  relative window (e.g. `30m`, `12h`, `3d`) covering that far back through
  now, instead of raw Unix-millisecond timestamps; omit it for no time
  filter. `--begin`/`--end` are an alternative, absolute way to specify
  the range (RFC3339 or `"YYYY-MM-DD HH:MM"`, e.g. `"2026-07-01 00:00"`) -
  either or both may be given (an omitted bound is unbounded), and they're
  mutually exclusive with `--last`. `--limit` (default 20) caps how many
  of the most recent entries come back - there's no `--page` to think
  about.
- Every command prints one JSON object to stdout:
  `{"status":200,"msg":"","data":...}` or `{"status":500,"msg":"..."}`, with
  exit code 0/1 to match.
