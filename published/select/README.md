# Select

The plugin and MCP server use the identifier `select`. Install Select as a new plugin when switching from `db-plugin`; saved connection settings use the same configuration folder.

A native Go MySQL MCP executable with a Wails desktop settings window. Organize connections as **Project → Environment → Database**. Teammates need neither Go nor Node.js.

## Use

Install Select from your marketplace or the shared release ZIP; its launcher selects the matching OS and processor. In a Codex or Claude session that runs MCP locally on your desktop, ask **“Open Select setup.”** The plugin opens a desktop window. Create a project, add an environment, and enter connection details. **Test connection** checks the current form without saving it and shows success or a specific error. Save the environment when ready; Save is enabled only for changed values and briefly confirms success. Return to your agent to query that project/environment.

Use the project and environment **⋯** menus to rename or delete saved entries. The project menu also offers preset colors. Renaming preserves saved connections and passwords; deleting removes settings from Select after confirmation and does not modify a database.

Passwords are stored as plaintext alongside each connection in your local `profiles.json`. On macOS/Linux the file is written with owner-only permissions (`0600`). The settings window and MCP status never return saved passwords. Keep this file private and out of repositories or shared packages. Query results return to the AI client. Use MySQL accounts with read-only grants and the required network/VPN access.

The executable has three modes:

- `select mcp`: stdio MCP server; no window opens automatically.
- `select setup`: open/focus the project manager (also the default when launched without arguments).
- `select desktop`: the separate single-connection MCPB mode, using extension-managed connection settings.

The MCP tools are `db_setup()`, `db_status()`, and `mysql_query({project, environment, sql})`. Setup returns `{opened: true}` after the window is ready. Status reports saved configuration, not connectivity.

Queries accept one read-only SELECT/SHOW/DESCRIBE/EXPLAIN statement. Multiple statements, writes, locking reads, file exports, and SQL variable assignments are rejected. Queries also run in read-only transactions, with a 45-second timeout and bounded results (1,000 rows / 4 MiB, marked `truncated` when limited). TLS verifies the server certificate whenever enabled; a custom CA can be configured.

## Platforms

The shared release package includes macOS, Windows and Linux executables for x64 (`amd64`) and ARM64, plus manifests, skills, icon and notices. The launcher selects the platform on which the local MCP agent runs; a WSL session uses Linux. Windows ARM64 uses a small x64 launcher under Windows 11 emulation, then starts the native ARM64 executable. The lower-level packager can still create single-platform ZIPs and the separate single-connection MCPB.

- macOS uses the system WebKit engine.
- Windows uses WebView2. The WebView2 runtime must be installed.
- Linux uses GTK3/WebKitGTK 4.1 and a graphical desktop for setup. Minimal/headless Linux systems need those system components before using the full GUI-enabled executable.

The repository prepares these targets; compilation and actual host verification are separate. See [maintenance](docs/maintenance.md) for the recorded validation and build requirements.

## Saved settings

Saved v3 settings include `projects[project].environments[environment].password`, including an explicit empty string for accounts without a password. V1/v2 settings remain readable; legacy v1 profiles appear under `falcon`. If an older connection has no inline password, save its password once in setup. The runtime no longer reads or writes OS credential stores. See [architecture](docs/architecture.md).

## Development

Use Go and the target platform's native build dependencies:

```sh
go mod download
go run ./scripts check
go run ./scripts build
```

Builds go into `build/<os>-<arch>/`, never `releases/`. `go run ./scripts notices` refreshes third-party notices. Packaging is a separate explicit command described in [maintenance](docs/maintenance.md); do not run it until a release is requested.

For development in local Claude Cowork or ChatGPT/Codex desktop sessions, run `go run ./scripts dev` and install `build/dev/select-dev.zip` once. This machine-specific wrapper launches the executable in this repository. Local builds load `src/ui/` directly: CSS edits appear live; reopen setup to load HTML/JavaScript edits. For Go changes, run `go run ./scripts build`, close the existing settings window, and restart the MCP session or desktop app. Skills/manifest changes and moving the repository still require updating the wrapper. See [development setup](docs/maintenance.md#local-desktop-development).

Source is in `src/`, embedded frontend assets in `src/ui/`, native developer tooling in `scripts/`, agent instructions in `skills/`, and guides/examples/schemas in `docs/`. There is no Node server, browser-facing settings HTTP server, or npm runtime.
