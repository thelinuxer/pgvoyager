# Desktop Self-Update — Design

**Date:** 2026-06-09
**Status:** Approved (pending spec review)
**Scope:** Add in-app auto-update for the PgVoyager **desktop** edition.

## Goal

Let the desktop app update itself: detect a newer GitHub release (already
implemented), auto-download + verify the new binary in the background, then
prompt the user to restart into the new version. No manual download/replace
when the binary lives in a user-writable location.

## Decisions (locked during brainstorming)

| Decision | Choice |
|----------|--------|
| Update model | In-place self-replace; fall back to a manual-download message when the executable's directory is not writable by the running user |
| Editions | **Desktop only** (`pgvoyager-desktop-*`). The server binary stays manual. |
| Integrity | Download `SHA256SUMS` from the release; verify the asset hash before swapping |
| Trigger/UX | Auto-download in the background when an update is found, then prompt "Restart now" |
| Installer | Add a user-writable install option (`~/.local/bin`, no sudo) so self-update can work |

## Non-goals (YAGNI)

- Server/browser-edition self-update.
- Privilege escalation to write root-owned install paths (e.g. `/usr/local/bin`).
  Those installs get the manual-download fallback.
- Code signing / Sigstore. SHA256 catches corruption and partial downloads;
  full supply-chain signing is out of scope for now.
- Delta/patch updates. Full binary download only.

## Architecture

### 1. Edition tagging

`internal/version`:

```go
// Edition is set at build time via ldflags ("desktop" for the desktop
// wrapper, empty/"server" otherwise). Gates self-update.
var Edition = ""
```

- `Makefile` `desktop` + `desktop-dev` targets append `-X …/internal/version.Edition=desktop` to LDFLAGS.
- `release.yml` desktop build steps add the same ldflag.
- The apply endpoint refuses with `409` unless `Edition == "desktop"`.

### 2. Release pipeline (`release.yml`)

After all artifacts are built into `releases/`, add a step:

```bash
cd releases && sha256sum * > SHA256SUMS
```

Upload `SHA256SUMS` as a release asset alongside the binaries. The updater
fetches it to verify the downloaded binary.

### 3. Backend — `internal/selfupdate` package

A focused package, no HTTP/gin dependency, unit-testable.

- `AssetName() string` — `pgvoyager-desktop-{GOOS}-{GOARCH}` from `runtime`.
  Returns error for unsupported OS/arch combos.
- `Writable() (bool, error)` — preflight: try to create+remove a temp file in
  `dir(os.Executable())`. False → caller surfaces the manual fallback.
- `Download(ctx, version) (stagedPath string, err error)`:
  1. Resolve the asset download URL + `SHA256SUMS` URL for the target tag.
  2. Stream the asset to a temp file **in the exe's directory** (same
     filesystem → atomic rename later), name `.<exe>.update-<tag>`.
  3. Fetch `SHA256SUMS`, parse the line for `AssetName()`, compute the
     staged file's SHA256, compare. Mismatch → delete temp, error.
  4. `chmod 0755`. Return staged path.
- `Apply(stagedPath) error`:
  1. `rename(stagedPath, exePath)` — atomic, allowed over a running binary on
     Linux/macOS (the running process keeps the old inode).
  2. Spawn the **new** binary detached: `exec.Command(exePath)`,
     `SysProcAttr{Setpgid: true}`, clean environment with `PGVOYAGER_PORT=0`,
     `Start()` (no `Wait`), then release.
  3. Trigger shutdown of the current process so its Chrome window + server tear
     down: send `SIGTERM` to self (the desktop `main` already bridges
     SIGTERM → ctx-cancel → `chromelaunch.Run` closes Chrome → clean exit).

State (the staged path between download and apply) lives in the handler layer
as a small guarded struct, not package-global in `selfupdate`.

### 4. Handlers / routes (`internal/api`, `internal/handlers`)

- `POST /update/download`
  - If `version.Edition != "desktop"` → `409 {error:"self-update not supported for this build"}`.
  - If `!Writable()` → `200 {ready:false, writable:false}`.
  - Else run `Download`, store staged path, → `200 {ready:true, writable:true}`.
  - On download/verify error → `200 {ready:false, writable:true, error:"…"}`.
- `POST /update/apply`
  - Requires a staged path from a prior successful download → else `409`.
  - Write the `200 {restarting:true}` response, then in a goroutine after a
    short delay call `selfupdate.Apply`.

Response types added to the existing update-handler file.

### 5. Frontend (`Header.svelte`)

Extend the existing update-check flow:

- State: `updateState: 'idle' | 'downloading' | 'ready' | 'restarting' | 'manual'`.
- When `checkUpdate` returns `hasUpdate` and edition supports it → auto
  `POST /update/download` (background), set `downloading`.
  - `ready:true` → `ready`.
  - `writable:false` → `manual` (show the existing release-page link — this is
    the path the current root-owned install takes).
  - `error` → `manual` (link to releases as a safe fallback).
- Badge rendering by state:
  - `downloading` → version badge + small spinner, title "Downloading update…".
  - `ready` → **"Update ready — Restart now"** button → `POST /update/apply`,
    set `restarting`.
  - `restarting` → "Updating…" disabled; window relaunches on the new version.
  - `manual` → current `update-available` link to the release page.
- `client.ts`: add `updateApi.download()` and `updateApi.apply()`.

### 6. Installer — user-writable option (`packaging/linux/install.sh`)

- Honor `INSTALL_DIR`; if it is writable without sudo (e.g. `~/.local/bin`),
  `cp` without `sudo`. Only use `sudo` when the target needs it.
- Add a `--user` convenience: sets `INSTALL_DIR="$HOME/.local/bin"`.
- If the chosen user dir is not on `PATH`, print a warning with the line to add.
- README: document `./install.sh --user` as the path that enables auto-update.

(Windows `install.ps1` user-path option is out of scope for this iteration;
note it as follow-up.)

## Data flow

```
check (6h / on mount) ──hasUpdate──> POST /update/download
                                         │
                   writable? ── no ──> {ready:false,writable:false} ─> UI "manual" (release link)
                        │ yes
                   download asset + SHA256SUMS ─> verify hash
                        │ ok                         │ fail
                   stage beside exe            {ready:false,error} ─> UI "manual"
                        │
                   {ready:true} ─> UI "Restart now"
                        │ click
                   POST /update/apply ─> rename over exe ─> spawn new (detached)
                                         └─> SIGTERM self ─> old Chrome+server exit
                                                            new window opens on new version
```

## Error handling

- Download failure / hash mismatch → staged temp deleted; UI falls back to the
  manual release link. Never swap an unverified binary.
- Not writable → no download attempt past preflight; manual link.
- `Apply` rename failure → return error, do not SIGTERM (app keeps running on
  old version); UI shows manual link.
- Spawn-new failure → log; still on old binary file path, but the file was
  already replaced. Mitigation: spawn-new **before** SIGTERM; only SIGTERM if
  `Start()` succeeded. If spawn fails after rename, surface error and keep the
  current process running so the user can retry/restart manually.

## Testing

- **Unit (`internal/selfupdate`):**
  - `AssetName()` mapping for supported GOOS/GOARCH; error on unsupported.
  - SHA256 verify: matching hash passes; corrupted file fails and is deleted.
  - `Writable()` true in a temp dir, false in a read-only dir.
  - `SHA256SUMS` parsing picks the correct line for the asset.
- **Handler:** edition gate (`409` when not desktop); `writable:false` shape;
  apply-without-download → `409`.
- **Manual/local verification:** build a desktop binary tagged as an older
  version into `~/.local/bin`, run it, confirm it auto-downloads the current
  release, shows "Restart now", and relaunches on the new version after click.
- **E2E:** assert the "Restart now" state renders given a mocked
  `download` → `ready` response. The real swap+restart is covered by the manual
  test (Playwright can't follow a process self-restart cleanly).

## Security notes

- Only HTTPS GitHub URLs; asset + `SHA256SUMS` from the same release.
- Verify SHA256 before making the binary executable in its final location.
- `Apply` runs only for the desktop edition and only against `os.Executable()`'s
  own directory — no arbitrary path writes.
- This protects against corrupted/partial downloads, not a compromised release
  (that needs signing — noted as future work).
