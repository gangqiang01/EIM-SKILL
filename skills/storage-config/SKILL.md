---
name: storage-config
description: Configure and switch the DeviceLink repo server's file-storage backend - MinIO, AWS S3, Alibaba Cloud OSS, or Azure Blob (there's no raw-local-disk mode; "local" deployments use a bundled MinIO instance) - used for storing OTA packages, app icons, etc. Use this whenever the user asks to add, list, activate, or delete a storage backend via the eim-skill CLI.
---

# Repo storage backend configuration

Wraps m2m-management-edge's `StorageController` (the repo server behind
OTA packages, app icons, etc). Requires `eim-skill` built and configured
(see repo README) - talks to the repo server, reusing the main AppHub
login token.

## When to use this skill

- "What storage backends are configured?" → `storage list`
- "Point the repo at our MinIO/S3/OSS bucket" → `storage add-s3`
- "Point the repo at our Azure Blob container" → `storage add-azure`
- "Switch the active storage backend to X" → `storage activate`
- "Remove storage backend X" → `storage delete`

## Commands

```
eim-skill storage list
eim-skill storage add-s3 <platform> <endpoint-url> <bucket> <access-key> <secret-key>
eim-skill storage add-azure <container> <connection-string>
eim-skill storage activate <sid>
eim-skill storage delete <sid>
```

`<platform>` for `add-s3` is stored verbatim as the backend's type - the
server enforces no fixed enum, so pick a consistent label such as
`"minio"`, `"s3"`, or `"oss"`.

## Notes and caveats

- `storage list` never returns the secret key (the server doesn't return
  it on that endpoint) - there's no way to read a previously-set secret
  key back out through this API.
- `storage add-s3`/`add-azure` validate connectivity before saving - if
  the bucket/container/credentials are wrong, the command fails with the
  server's connection error rather than silently saving bad config.
- Only the first storage backend ever created becomes active
  automatically; use `storage activate` for any backend added afterward.
- `storage delete` also deletes every repo object (OTA packages, app
  icons, etc) stored in that backend - it's destructive, confirm with the
  user before running it.
