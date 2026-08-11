---
name: ota-deployment
description: Look up Android/Windows/Docker/Linux packages and generic files available in the DeviceLink repo server, check what's installed on a device, check a device's most recent OTA/BSP task status, and push an install/update/uninstall to a single device or a whole device group (OTA). Use this whenever the user asks to deploy, install, update, uninstall, or OTA a package/app/file to DeviceOn-EIM-managed devices, or asks whether a recent deployment succeeded, via the eim-skill CLI. 中文触发词：OTA、软件部署、应用部署、安装包、批量部署、按组部署、卸载应用、部署文件、部署状态、安装状态、BSP、系统更新。
---

# OTA deployment

Wraps the DeviceLink web console's OTA flow, generalized across every
package type it supports: Android APKs, Windows exe installers, Windows
zip/compressed packages, Docker compose/swarm packages, Linux deb/tar
packages, generic files, and (status-only) BSP/system-image updates.
Requires `eim-skill` built and configured (see repo README) - talks to
both the main DeviceOn-EIM server and its companion repo server
(m2m-management-edge).

## The workflow (same shape for every type)

1. **`ota <type> list`** - what's available in the repo server to push.
2. **`ota <type> installed <endpoint-or-name>`** - what's already on the
   device (Android/Windows/Linux only - Docker has no "list everything
   deployed" query at all: `device docker monitored` only reports
   containers currently selected for monitoring, not every deployed
   container; File has no "installed" concept).
3. **`ota <type> apply`/`install`/`upgrade` `<endpoint-or-name-or-group>
   <name> <version>`** - look up that name+version in the repo listing,
   resolve its download path, and push it.
4. **`ota <type> status <endpoint-or-name>`** - check that device's most
   recent OTA task status for this package type (queued/running/success/
   failed) - see the caveats below, this is less battle-tested than
   list/installed/apply.

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

- "What Android apps/Windows packages/Docker images/Linux packages can we push?" / "查看可部署的安装包" → `ota <type> list` (i.e. `ota linux list`/`ota android list`/`ota windows-exe list`/`ota windows-zip list`/`ota docker list` - NOT `ota file list --type linux`/`windows`/`android`, which is a different, unrelated command; see caveat below)
- "What's installed on device X?" → `ota <type> installed <endpoint-or-name>`
- "Deploy/update package P version V on device X" / "给设备X部署/更新应用P" → `ota <type> apply` (or `install`/`upgrade` for Windows)
- "Roll out package P version V to every device in group G" / "给设备组G批量部署应用P" → `ota <type> apply-group` (or `install-group`/`upgrade-group`)
- "Uninstall app/package P from device X" / "卸载设备X上的应用P" → `ota android uninstall` / `ota windows-exe uninstall` / `ota windows-zip uninstall` / `ota docker uninstall`
- "Push this file to device X" / "部署文件到设备X" → `ota file deploy`
- "Did the last OTA/deploy on device X succeed?" / "设备X最近一次部署/安装状态怎么样" → `ota <type> status <endpoint-or-name>`
- "What's device X's BSP/system update status?" / "设备X的BSP/系统更新状态" → `ota bsp status <endpoint-or-name>`

## Commands

```
../scripts/eim-skill ota linux list
../scripts/eim-skill ota linux installed <endpoint-or-name>
../scripts/eim-skill ota linux status <endpoint-or-name>
../scripts/eim-skill ota linux apply <endpoint-or-name> <product-name> <version>
../scripts/eim-skill ota linux apply-group <group-name> <product-name> <version> [--label <name>]

../scripts/eim-skill ota android list
../scripts/eim-skill ota android installed <endpoint-or-name>
../scripts/eim-skill ota android status <endpoint-or-name>
../scripts/eim-skill ota android apply <endpoint-or-name> <pkg-name> <version>
../scripts/eim-skill ota android apply-group <group-name> <pkg-name> <version> [--label <name>]
../scripts/eim-skill ota android uninstall <endpoint-or-name> <pkg-name>

../scripts/eim-skill ota windows-exe list
../scripts/eim-skill ota windows-exe installed <endpoint-or-name>
../scripts/eim-skill ota windows-exe status <endpoint-or-name>
../scripts/eim-skill ota windows-exe install <endpoint-or-name> <project-name> <version>
../scripts/eim-skill ota windows-exe install-group <group-name> <project-name> <version>
../scripts/eim-skill ota windows-exe upgrade <endpoint-or-name> <project-name> <version>
../scripts/eim-skill ota windows-exe upgrade-group <group-name> <project-name> <version>
../scripts/eim-skill ota windows-exe uninstall <endpoint-or-name> <project-name> <version> [--tool <tool>]

../scripts/eim-skill ota windows-zip list
../scripts/eim-skill ota windows-zip installed <endpoint-or-name>
../scripts/eim-skill ota windows-zip status <endpoint-or-name>
../scripts/eim-skill ota windows-zip install <endpoint-or-name> <project-name> <version> [--parameter <args>]
../scripts/eim-skill ota windows-zip install-group <group-name> <project-name> <version> [--parameter <args>] [--label <name>]
../scripts/eim-skill ota windows-zip update <endpoint-or-name> <project-name> <version> [--parameter <args>]
../scripts/eim-skill ota windows-zip update-group <group-name> <project-name> <version> [--parameter <args>] [--label <name>]
../scripts/eim-skill ota windows-zip uninstall <endpoint-or-name> <project-name> <version>

../scripts/eim-skill ota docker list
../scripts/eim-skill ota docker status <endpoint-or-name>
../scripts/eim-skill ota docker apply <endpoint-or-name> <tag> <version>
../scripts/eim-skill ota docker apply-group <group-name> <tag> <version> [--label <name>]
../scripts/eim-skill ota docker uninstall <endpoint-or-name> <tag> <version>

../scripts/eim-skill ota file list --type <type>
../scripts/eim-skill ota file status <endpoint-or-name> --type <type>
../scripts/eim-skill ota file deploy <endpoint-or-name> <file-name> <save-path> --type <type> [--exec]
../scripts/eim-skill ota file deploy-group <group-name> <file-name> <save-path> --type <type> [--exec] [--label <name>]

../scripts/eim-skill ota bsp status <endpoint-or-name>
```

`apply`/`install`/`upgrade`/`deploy` all look up `<name>`/`<version>` in
the repo server's package listing to resolve the download path, so the
name/version must exactly match an entry from the corresponding `list`
command.

## Notes and caveats

- Android/Docker/Linux each have **one** device-side resource for both
  install and update (the server distinguishes by whether the package is
  already present) - `apply` covers both cases. Windows genuinely has
  **separate** install vs. upgrade resources, so it's exposed as distinct
  `install`/`upgrade` commands rather than one collapsed `apply`.
- `windows-exe uninstall`'s `--tool` defaults to `"Inno Setup"` when
  omitted - pass `--tool "WiX Toolset"` explicitly if that's how the
  package was originally installed. `install`/`upgrade` read it from the
  repo listing automatically either way, no flag needed.
- `ota file list`/`deploy`/`deploy-group` require `--type` - there's no
  unconditional "list all files" endpoint; `--type` must match whatever
  category the file was uploaded under in the repo server. **This `--type`
  is a freeform label, not an OS/platform selector** - `ota file list
  --type linux` (or `windows`/`android`) reliably returns an empty list
  (confirmed via a real bug report: a user tried exactly this, expecting
  it to return Linux/Windows/Android packages) unless someone happened to
  upload a generic file under that literal category name. For actual
  platform packages, use `ota linux list`/`ota android list`/`ota
  windows-exe list`/`ota windows-zip list`/`ota docker list` instead -
  `ota file` is a completely separate, generic-file-only feature.
- Repo-listing endpoints (`list` for all six types) and the full
  device-resolve → repo-lookup → error-handling pipeline are verified
  against a real server. The final device-side deploy call for each type
  is implemented from AppHub-v2.3.0.md's documented request/response
  shapes but wasn't exercised against a live device.
- `ota <type> status` (all types including `bsp`) is **not documented in
  AppHub-v2.3.0.md** - found via an apifox-exported OpenAPI spec of
  DeviceOn-EIM's own web console. Every `status` command was confirmed to
  return a successful (non-error) response against a live server, but
  that server had zero OTA/BSP task history recorded, so no command's
  *non-empty* response shape has been verified against real data. Beyond
  that: `linux`/`android`/`windows-exe`/`windows-zip`/`docker` share one
  backend ("appstatusmgr") keyed by a "type" business-classifier whose
  value is only documented for Android (the fixed literal `"sg"`) - these
  five commands all reuse `"sg"` on the assumption it's shared across the
  whole manager, which is unverified for anything but Android. `file`'s
  status uses a separate backend ("filestatusmgr") keyed by the same
  `--type` category as `ota file list`/`deploy`. `bsp` uses its own
  backend ("deploymgr") with no type parameter at all, so it carries the
  least additional risk beyond "shape of a populated response is
  unconfirmed". Verify against your own server (ideally after triggering
  a real deploy) before relying on any of these in an unattended workflow.
