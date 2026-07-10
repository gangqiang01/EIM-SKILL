---
name: ota-deployment
description: Look up Linux software packages available in the DeviceLink repo server, check what's installed on a device, and push an install/update to a single device or a whole device group (OTA). Use this whenever the user asks to deploy, update, or OTA a Linux package to AppHub-managed devices via the eim-skill CLI.
---

# OTA deployment (Linux packages)

Wraps the DeviceLink web console's Linux app-control OTA flow. Requires
`eim-skill` built and configured (see repo README) - this feature talks to
both the main AppHub server and its companion repo server
(m2m-management-edge).

## When to use this skill

- "What packages can we push to devices?" → `ota list-available`
- "What's installed on device X?" → `ota list-installed`
- "Update/install package P version V on device X" → `ota apply`
- "Roll out package P version V to every device in group G" → `ota apply-group`

## Commands

```
eim-skill ota list-available
eim-skill ota list-installed <endpoint>
eim-skill ota apply <endpoint> <product-name> <version>
eim-skill ota apply-group <group-name> <product-name> <version> [--label <label-name>]
```

`apply`/`apply-group` look up `<product-name>`/`<version>` in the repo
server's package listing to resolve the download path, then push it via
the same devicecontrol resource the web console uses - so the version
string must exactly match an entry from `ota list-available`.

## Notes and caveats

- Linux devices/packages only for now - Windows (exe/zip), Android (APK),
  HarmonyOS, and Docker each use different repo endpoints and
  devicecontrol resources that aren't wired up in this CLI.
- `apply`/`apply-group` don't remove anything first; if the target package
  is already installed at a different version, the same call updates it
  in place (install and update use the identical server-side action).
