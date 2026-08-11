---
name: sub-devices
description: Query sub-devices ("子设备") registered under a parent DeviceOn-EIM device via the ithings/twin server - list them, see which services/properties a sub-device has and their current real-time values, get historical data for one property, and list recorded events. Use this whenever the user asks about a sub-device, digital twin, or Modbus/SNMP-connected "thing" attached to a device, via the eim-skill CLI. 中文触发词：子设备、数字孪生、设备孪生、子设备属性、子设备服务、子设备事件、子设备历史数据。
---

# Sub-devices

Wraps DeviceOn-EIM's sub-device/digital-twin service (`apphub-ithings`) -
a separate, read-only query surface from the main device-management API,
for things (Modbus/SNMP/executor/restful-connected) registered under a
parent device. Requires `eim-skill` built and configured (see repo
README) - talks to a third server (the ithings server, its own base URL)
beyond the main DeviceOn-EIM server and the repo server.

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

- "What sub-devices does device X have?" / "查看设备X下的子设备" → `subdevice list --parent X`
- "What services/properties does this sub-device have, and what are their current values?" / "查看子设备的服务和属性/实时值" → `subdevice get`
- "What's the current value of property P (under service S) on this sub-device?" / "查看子设备指定service/属性的当前值" → `subdevice get --service S --property P`
- "Show me the history of property P on this sub-device" / "查看子设备P属性的历史数据" → `subdevice history`
- "What events has this sub-device recorded?" / "查看子设备的事件记录" → `subdevice events`

## Commands

```
../scripts/eim-skill subdevice list [--parent <endpoint-or-name>]

../scripts/eim-skill subdevice get <device-id-or-name> [--service <name>] [--property <name>]

../scripts/eim-skill subdevice history <device-id-or-name> <service-name> <property-name>
                           [--last <30m|12h|3d>] [--begin <time>] [--end <time>] [--count N]

../scripts/eim-skill subdevice events <device-id-or-name> [--page N] [--limit N]
                           [--last <30m|12h|3d>] [--begin <time>] [--end <time>]
```

`<device-id-or-name>` accepts either the sub-device's id or its name -
resolved against the live sub-device list. Unlike parent devices'
endpoint/devName, a sub-device's name is **not** guaranteed unique across
different parents - an ambiguous name errors out listing every matching
candidate (with its parent) rather than guessing which one you meant; use
the device id instead if that happens.

## Notes and caveats

- This is a genuinely separate backend/server from the main
  device-management API - its own base URL
  (`--ithings-base-url`/`ithings_base_url` config key, default
  `http://127.0.0.1:8082`), though it accepts the same `accesstoken`
  issued by the main server's `/login` (confirmed against a live server -
  no separate ithings login).
- `subdevice list` has no server-side filter - `--parent` is applied
  client-side after fetching the full sub-device list.
- `subdevice get` (the "twin") doubles as service/property discovery: it
  returns every service+property the sub-device has ever reported, each
  with its current real-time value. There's no separate schema-only
  endpoint wrapped here, so this is how you find the exact
  `service-name`/`property-name` strings (the response's `svc`/`pn`
  fields) to pass to `subdevice history`. An offline or
  never-reported sub-device returns an empty list rather than an error
  (confirmed against a live server - the server's own raw response is a
  bare JSON `null` in that case, normalized to `[]` here). `--service`/
  `--property` narrow this down client-side (the underlying endpoint has
  no filter for either, always returning every reported property) - with
  both given, the single matching property is returned directly (not
  wrapped in a list), or an error if that exact service+property
  combination has no current value; with just one of the two, every match
  comes back as a list.
- A property's `val` is left as raw/untyped JSON rather than forced into
  one Go type - AppHub-v2.3.0.md's own examples show it as a plain number
  in one response and a numeric *string* in another for the same shape,
  so nothing safe can be assumed about its type beyond "valid JSON".
- **`subdevice history`'s `--last`/`--begin`/`--end`/`--count` are
  confirmed, via live testing, to have no observable filtering effect** -
  every range tried (including deliberately absurd ones, like a
  year-2000 or year-2100 window) returned the identical most-recent ~20
  samples. They're still sent, matching both AppHub-v2.3.0.md's
  documented request shape and a real packet capture, in case a
  different server version honors them - but don't rely on them actually
  narrowing today's result set; filter client-side if you need a real
  window.
- `subdevice events` (`GET /v1/deviceEvent/byPage`) is not documented in
  AppHub-v2.3.0.md at all - confirmed against a live server via a real
  packet capture to respond successfully, but no sub-device in the
  environment tested against had any recorded events, so a populated
  response's per-event shape (beyond the `{"list":[...],"total":N}`
  wrapper) is unconfirmed - entries are passed through as raw JSON rather
  than reshaped into a typed struct. Its time filters are likewise
  untested (no data existed to verify filtering against, one way or the
  other).
- Every command prints one JSON object to stdout:
  `{"status":200,"msg":"","data":...}` or `{"status":500,"msg":"..."}`, with
  exit code 0/1 to match.
