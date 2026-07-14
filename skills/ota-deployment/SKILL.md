---
name: ota-deployment
description: Look up Android/Windows/Docker/Linux packages and generic files available in the DeviceLink repo server, check what's installed on a device, and push an install/update/uninstall to a single device or a whole device group (OTA). Use this whenever the user asks to deploy, install, update, uninstall, or OTA a package/app/file to AppHub-managed devices via the eim-skill CLI. 中文触发词：OTA、软件部署、应用部署、安装包、批量部署、按组部署、卸载应用、部署文件。
---

# OTA deployment

Wraps the DeviceLink web console's OTA flow, generalized across every
package type it supports: Android APKs, Windows exe installers, Windows
zip/compressed packages, Docker compose/swarm packages, Linux deb/tar
packages, and generic files. Requires `eim-skill` built and configured
(see repo README) - talks to both the main AppHub server and its
companion repo server (m2m-management-edge).

## The workflow (same shape for every type)

1. **`ota <type> list`** - what's available in the repo server to push.
2. **`ota <type> installed <endpoint-or-name>`** - what's already on the
   device (Android/Windows/Linux only - Docker's equivalent is
   `device docker info`; File has no "installed" concept).
3. **`ota <type> apply`/`install`/`upgrade` `<endpoint-or-name-or-group>
   <name> <version>`** - look up that name+version in the repo listing,
   resolve its download path, and push it.

## When to use this skill

- "What Android apps/Windows packages/Docker images/Linux packages can we push?" / "查看可部署的安装包" → `ota <type> list`
- "What's installed on device X?" → `ota <type> installed <endpoint-or-name>`
- "Deploy/update package P version V on device X" / "给设备X部署/更新应用P" → `ota <type> apply` (or `install`/`upgrade` for Windows)
- "Roll out package P version V to every device in group G" / "给设备组G批量部署应用P" → `ota <type> apply-group` (or `install-group`/`upgrade-group`)
- "Uninstall app/package P from device X" / "卸载设备X上的应用P" → `ota android uninstall` / `ota windows-exe uninstall` / `ota windows-zip uninstall` / `ota docker uninstall`
- "Push this file to device X" / "部署文件到设备X" → `ota file deploy`

## Commands

```
eim-skill ota linux list
eim-skill ota linux installed <endpoint-or-name>
eim-skill ota linux apply <endpoint-or-name> <product-name> <version>
eim-skill ota linux apply-group <group-name> <product-name> <version> [--label <name>]

eim-skill ota android list
eim-skill ota android installed <endpoint-or-name>
eim-skill ota android apply <endpoint-or-name> <pkg-name> <version>
eim-skill ota android apply-group <group-name> <pkg-name> <version> [--label <name>]
eim-skill ota android uninstall <endpoint-or-name> <pkg-name>

eim-skill ota windows-exe list
eim-skill ota windows-exe installed <endpoint-or-name>
eim-skill ota windows-exe install <endpoint-or-name> <project-name> <version>
eim-skill ota windows-exe install-group <group-name> <project-name> <version>
eim-skill ota windows-exe upgrade <endpoint-or-name> <project-name> <version>
eim-skill ota windows-exe upgrade-group <group-name> <project-name> <version>
eim-skill ota windows-exe uninstall <endpoint-or-name> <project-name> <version> --tool <tool>

eim-skill ota windows-zip list
eim-skill ota windows-zip installed <endpoint-or-name>
eim-skill ota windows-zip install <endpoint-or-name> <project-name> <version> [--parameter <args>]
eim-skill ota windows-zip install-group <group-name> <project-name> <version> [--parameter <args>] [--label <name>]
eim-skill ota windows-zip update <endpoint-or-name> <project-name> <version> [--parameter <args>]
eim-skill ota windows-zip update-group <group-name> <project-name> <version> [--parameter <args>] [--label <name>]
eim-skill ota windows-zip uninstall <endpoint-or-name> <project-name> <version>

eim-skill ota docker list
eim-skill ota docker apply <endpoint-or-name> <tag> <version>
eim-skill ota docker apply-group <group-name> <tag> <version> [--label <name>]
eim-skill ota docker uninstall <endpoint-or-name> <tag> <version>

eim-skill ota file list --type <type>
eim-skill ota file deploy <endpoint-or-name> <file-name> <save-path> --type <type> [--exec]
eim-skill ota file deploy-group <group-name> <file-name> <save-path> --type <type> [--exec] [--label <name>]
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
- `windows-exe uninstall` needs `--tool` ("Inno Setup" or "WiX Toolset",
  matching how the package was originally installed) - `install`/`upgrade`
  read it from the repo listing automatically, no flag needed.
- `ota file list`/`deploy`/`deploy-group` require `--type` - there's no
  unconditional "list all files" endpoint; `--type` must match whatever
  category the file was uploaded under in the repo server.
- Repo-listing endpoints (`list` for all six types) and the full
  device-resolve → repo-lookup → error-handling pipeline are verified
  against a real server. The final device-side deploy call for each type
  is implemented from AppHub-v2.3.0.md's documented request/response
  shapes but wasn't exercised against a live device.
