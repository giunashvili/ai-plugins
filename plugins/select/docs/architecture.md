# Architecture

The same executable runs either an MCP server (`mcp`), the Wails settings window (`setup`), or a single native desktop-extension connection (`desktop`). No Node process or upstream npm MCP server is spawned.

The official Go MCP SDK handles stdio protocol messages. The MySQL driver connects directly over TCP. The MySQL AST parser restricts queries to one read statement, rejects executable SQL comments and side-effecting read forms, and MySQL read-only transactions enforce the database boundary. Read-only grants remain required. Connections and rows are closed on completion, cancellation, or failure. Results are JSON text with `rows`, `rowCount`, and `truncated`.

## Settings window

HTML/CSS/JavaScript is embedded using Go's embed package for distribution. Builds launched from the repository's `build/<os>-<arch>/` directory load `src/ui/` directly for development; `SELECT_UI_DIR` is an explicit override. Asset responses disable caching. In development, the page checks the stylesheet once per second while visible and refreshes it without resetting the form; reopening setup loads HTML/JavaScript edits. The page calls `ListProjects`, `CreateProject`, `SaveEnvironment`, `TestConnection`, and the boolean `DevelopmentMode` through Wails bindings. Passwords are write-only: no bound method returns a saved password. The page loads no remote content and exposes no browser HTTP API.

`TestConnection` validates the current form with the same password-reuse behavior as saving, then runs `SELECT 1` through the normal read-only query path with a 15-second overall timeout. It never writes configuration or returns credentials. Connection errors use the existing sanitized diagnostics. The UI tracks actual form values against the loaded baseline: edits show Unsaved changes, reverting disables Save again, and a successful save briefly shows Saved. Testing has its own inline result and does not clear the dirty state. Form controls are disabled while a save or test is in progress.

Project and environment menus bind `RenameProject`, `DeleteProject`, `RenameEnvironment`, `DeleteEnvironment`, and `SetProjectColor`. These operations serialize configuration changes under the same mutex and atomic file replacement as saving a connection; they do not connect to MySQL. Renames move the complete saved entry, preserving inline credentials, TLS settings and project color, and reject duplicate destination names. Renaming or choosing a color preserves the current connection draft in the UI. Delete dialogs identify the selected entry and explain that only saved Select settings are removed. Deleting a project removes all its saved environments; deleting the last entry returns the UI to the appropriate empty state.

Each project may have an optional `color` value: `blue`, `green`, `purple`, `orange`, `red`, `pink`, `teal`, `amber`, or an empty/default value. These are preset choices, never arbitrary CSS. Colors appear on the sidebar folder icon and project heading. Older files without a color continue to load.

`db_setup` launches the same executable in setup mode with a private stdout readiness pipe. The window acknowledges readiness only after its DOM loads. Its output never shares MCP stdout. A per-config-directory OS lock allows one settings instance. A short-lived random token in an owner-private session file authorizes loopback TCP focus messages (not HTTP and never carrying credentials). Repeated requests focus the existing window. The MCP process continues working when setup is open; settings are loaded fresh per query.

## Storage and migration

Project and environment names support capitals, spaces, punctuation and Unicode. Names must contain visible text and no ASCII control characters; they are JSON map keys, never filesystem paths. Frontend maps use null prototypes so names such as `__proto__` and `constructor` remain ordinary keys.

The displayed product name is Select; the plugin/MCP identifier and executable name are `select`. This is a new plugin identity compared with `db-plugin`, so install Select as a new plugin and disable the old one to avoid duplicate tools. Environment variables and configuration paths retain their existing names so saved connections keep working.

V3 JSON remains `profiles.json`, with connection settings and a plaintext `password` at `projects[project].environments[environment]`. An absent password means it needs to be entered; an empty string is a saved empty password. Writes use a temporary file, owner-only permissions (`0600` on macOS/Linux), and atomic replacement. Windows uses the configuration directory's inherited ACLs; Go file modes do not configure Windows ACLs. Deprecated `runtimeDir` is accepted on reads and omitted on new writes. This file contains credentials and must not be shared or committed.

- macOS: `~/Library/Application Support/DB Plugin`
- Windows: `%APPDATA%\DB Plugin`
- Linux: `$XDG_CONFIG_HOME/db-plugin`, or `~/.config/db-plugin`

`DB_PLUGIN_CONFIG_DIR` overrides the location; `FALCON_DB_CONFIG_DIR` remains a compatibility override. Without an override, a missing current file falls back to the old Falcon DB location. V1 profiles map only to project `falcon`. Reads do not alter old files; saves use the configured current directory.

V1/v2 settings remain readable and are written as v3 on the next save. The runtime has no Keychain, Credential Manager, Secret Service or keyring dependency and never imports credentials automatically. Older connections without an inline password need it saved again in setup. Existing OS credential entries are not deleted. `ListProjects` strips passwords before returning connection metadata to the settings window, and `db_status` exposes only project/environment names. Saving with no new password preserves the selected environment's existing password; new environments with a blank password save an explicit empty string.
