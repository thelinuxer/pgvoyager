# Desktop Self-Update — Design

**Date:** 2026-06-09
**Status:** Approved (pending spec review)
**Scope:** Add in-app auto-update for the PgVoyager **desktop** edition,
driven by the desktop Go process (not the web frontend).

## Goal

Let the desktop app update itself: the desktop process checks GitHub for a
newer release on its own timer, downloads + SHA256-verifies the new binary in
the background, and stages it. The frontend only *reflects* that state and
offers a "Restart now" action. On restart the process swaps itself and
relaunches into the new version. No manual download/replace when the binary
lives in a user-writable location.

## Decisions (locked during brainstorming)

| Decision | Choice |
|----------|--------|
| Who drives it | **Desktop Go process** owns check + download + verify + stage. Frontend is a thin status view + restart button. |
| Update model | In-place self-replace; fall back to a manual-download message when the executable's directory is not writable by the running user |
| Editions | **Desktop only** (`pgvoyager-desktop-*`). The server binary keeps today's frontend-only check (shows a release link). |
| Integrity | Download `SHA256SUMS` from the release; verify the asset hash before staging |
| Trigger/UX | Background auto-download when an update is found, then prompt "Restart now" |
| Apply trigger | Desktop-only guarded `POST /update/restart` (OriginGuard + edition gate); not present on the server binary |
| Installer | Add a user-writable install option (`~/.local/bin`, no sudo) so self-update can work |

## Why backend-driven (not frontend-driven)

- The desktop app is already a long-lived Go process managing the window — it
  can check/download on a ticker with no SPA loaded and no JS timers.
- A frontend-issued apply would race the window/server teardown it triggers.
- The swap+re-exec primitive must **not** be reachable as a generic mutating
  HTTP route from rendered web content. Keeping it in-process (and the one
  trigger endpoint desktop-only + guarded) shrinks the attack surface.
- The headless server binary never gets a "replace my binary" route.

## Privileged update (amendment 2026-06-09)

Root-owned installs (`/usr/local/bin`) now self-update too, by elevating ONLY
the binary-swap step through a GUI auth dialog:

- **Linux:** `pkexec` (polkit) — implemented now.
- **macOS:** `osascript … with administrator privileges` — platform seam, stub.
- **Windows:** UAC `runas` — platform seam, stub.

Mechanics:
- Staging goes to the exe's dir when writable, else to a user-writable temp
  dir (`os.TempDir()/pgvoyager-update`).
- `Apply` tries `os.Rename`; on failure (not writable / cross-device) and when
  elevation is available, it runs an elevated copy+chmod of the staged file
  over the exe. The new process is spawned as the normal user.
- A non-writable install is `ready` (not `manual`) when elevation is available;
  `State.NeedsElevation=true` lets the UI hint that an admin password dialog
  will appear. If elevation is unavailable too, it falls back to `manual`.
- Cancelling the password dialog returns an error → status stays `ready`.

## Non-goals (YAGNI)

- Server/browser-edition self-update.
- macOS/Windows privileged update beyond the platform seam (stubbed for now).
- Code signing / Sigstore (SHA256 catches corruption, not a compromised
  release — future work).
- Delta/patch updates.

## Architecture

### 1. Edition tagging

`internal/version`:

```go
// Edition is set at build time via ldflags ("desktop" for the desktop
// wrapper, empty otherwise). Gates self-update behavior.
var Edition = ""
```

- `Makefile` `desktop` + `desktop-dev` targets append
  `-X …/internal/version.Edition=desktop` to LDFLAGS.
- `release.yml` desktop build steps add the same ldflag.

### 2. Release pipeline (`release.yml`)

After artifacts are built into `releases/`, add:

```bash
cd releases && sha256sum * > SHA256SUMS
```

Upload `SHA256SUMS` as a release asset.

### 3. `internal/selfupdate` — mechanism (no gin dependency)

Pure functions + a `Manager`, unit-testable.

Functions:
- `AssetName() (string, error)` — `pgvoyager-desktop-{GOOS}-{GOARCH}`; error on
  unsupported OS/arch.
- `Writable() (bool, error)` — try create+remove a temp file in
  `dir(os.Executable())`.
- `download(ctx, tag) (stagedPath string, err error)` — stream the asset to a
  temp file **in the exe's directory** (`.<exe>.update-<tag>`), fetch + parse
  `SHA256SUMS`, verify the staged file's hash, `chmod 0755`. On any failure,
  delete the temp file and return error.
- `apply(stagedPath) error` — `rename(staged, exePath)` (atomic, allowed over a
  running binary), spawn the **new** binary detached (`SysProcAttr{Setpgid:true}`,
  clean env with `PGVOYAGER_PORT=0`, `Start()` then release), then **only if
  Start succeeded** send `SIGTERM` to self. The desktop `main` already bridges
  SIGTERM → ctx-cancel → `chromelaunch.Run` closes Chrome → clean exit.

`Manager` (lives in the desktop process):
- Holds `state` under a mutex: `status` ∈ `idle | checking | downloading |
  ready | error | manual | unsupported`, plus `current`, `latest`,
  `stagedPath`, `releaseURL`, `errMsg`.
- `Start(ctx, interval)` — runs an immediate check then a ticker loop.
- Each cycle: fetch latest release (reuses the existing GitHub fetch); if newer:
  preflight `Writable()` → false sets `manual` (with `releaseURL`); else set
  `downloading`, call `download`, set `ready` (store `stagedPath`) or `error`.
- `Status() State` — snapshot for the status endpoint.
- `Restart() error` — guard `status==ready` + non-empty `stagedPath`, then
  `apply`.

### 4. Routes / handlers

Shared (`api.RegisterRoutes`, both editions):
- `GET /update/status` — returns a unified shape:
  - Desktop: `{edition:"desktop", status, currentVersion, latestVersion,
    releaseUrl}` from the injected `Manager`.
  - Server (no manager): `{edition:"server", status: hasUpdate?"manual":"idle",
    currentVersion, latestVersion, releaseUrl}` computed from the existing
    check logic (preserves today's behavior).
- `GET /update/check` — kept for backward compatibility / server edition.

Desktop-only (registered in `cmd/desktop` after `api.RegisterRoutes`, bound to
the `Manager`):
- `POST /update/restart` — `409` unless `version.Edition=="desktop"` and status
  is `ready`; responds `{restarting:true}`, then calls `Manager.Restart()` in a
  goroutine after a short delay. Sits behind the existing OriginGuard.

Injection: `cmd/desktop/main` constructs the `Manager`, starts it, and provides
it to the router (status handler + restart route). The shared status handler
reads the manager via a setter/closure; when nil (server binary) it falls back
to the compute-from-check path.

### 5. Frontend (`Header.svelte`, `client.ts`)

- Replace the direct `checkUpdate` call with polling `GET /update/status`
  (on mount + a modest interval, e.g. 30 min — the heavy lifting is now
  server-side; polling is just to learn when `ready`).
- Render by `status`:
  - `downloading` → version badge + spinner, title "Downloading update…".
  - `ready` → **"Update ready — Restart now"** button → `POST /update/restart`,
    set local `restarting`.
  - `restarting` → "Updating…" disabled; window relaunches on the new version.
  - `manual` → current `update-available` link to `releaseUrl` (root-owned and
    server-edition installs land here).
  - `idle | checking | error` → plain version badge.
- `client.ts`: `updateApi.status()` and `updateApi.restart()`.

### 6. Installer — user-writable option (`packaging/linux/install.sh`)

- Honor `INSTALL_DIR`; if writable without sudo, `cp` without `sudo`. Use
  `sudo` only when the target needs it.
- Add `--user` → `INSTALL_DIR="$HOME/.local/bin"`.
- If the chosen user dir is not on `PATH`, warn with the line to add.
- README: document `./install.sh --user` as the path that enables auto-update.
- Windows `install.ps1` user-path option: follow-up, out of scope here.

## Data flow

```
desktop main starts Manager.Start(ctx, interval)
        │
   ticker cycle: fetch latest release
        │ newer?
        ├─ no  ─> status: idle
        └─ yes ─> Writable()?
                    ├─ no  ─> status: manual (releaseUrl)
                    └─ yes ─> status: downloading
                               download asset + SHA256SUMS ─> verify
                                  ├─ fail ─> status: error (then retry next cycle)
                                  └─ ok   ─> stage beside exe; status: ready

frontend GET /update/status (poll) ─> renders badge / "Restart now"
        │ click
   POST /update/restart (desktop-only, guarded) ─> Manager.Restart()
        └─ apply: rename over exe ─> spawn new (detached) ─> SIGTERM self
                                     old Chrome+server exit; new window opens
```

## Error handling

- Download / hash mismatch → staged temp deleted; `status:error`; retried on the
  next ticker cycle. Never stage an unverified binary.
- Not writable → no download; `status:manual`.
- `apply` rename failure → return error, do not SIGTERM (stays on old version);
  status reflects error; user can retry.
- Spawn-new failure → because spawn happens **before** SIGTERM, the process
  keeps running (on the already-replaced file path); surface the error so the
  user can restart manually. (The new file is valid and verified; a manual
  relaunch picks it up.)

## Testing

- **Unit (`internal/selfupdate`):**
  - `AssetName()` mapping for supported GOOS/GOARCH; error on unsupported.
  - SHA256 verify: matching passes; corrupted fails and temp is deleted.
  - `SHA256SUMS` parsing selects the correct line for the asset.
  - `Writable()` true in temp dir, false in a read-only dir.
  - `Manager` state transitions with a stubbed release fetch + stubbed
    download (newer→downloading→ready; not-writable→manual; verify-fail→error).
- **Handler:** `GET /update/status` shape for desktop (manager) vs server (nil
  manager); `POST /update/restart` → `409` when not desktop or not ready.
- **Manual/local verification:** build a desktop binary tagged as an *older*
  version into `~/.local/bin`, run it, confirm the process auto-downloads the
  current release, status flips to `ready`, the button appears, and clicking it
  relaunches on the new version.
- **E2E:** assert the "Restart now" state renders given a mocked
  `GET /update/status` = `ready`. Real swap+restart is covered by the manual
  test (Playwright can't follow a process self-restart cleanly).

## Security notes

- Check/download/verify run in-process; the only mutating route
  (`/update/restart`) exists solely on the desktop binary, behind OriginGuard +
  edition gate, and only signals applying an already-verified staged update.
- Only HTTPS GitHub URLs; asset + `SHA256SUMS` from the same release.
- Verify SHA256 before the binary reaches its final executable location.
- `apply` targets only `os.Executable()`'s own directory — no arbitrary paths.
- Protects against corrupted/partial downloads, not a compromised release
  (signing is future work).
