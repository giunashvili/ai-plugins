# Native development and validation

Select lives at `plugins/select/` in the `ai-plugins` repository. Run all commands in this guide from that directory. Marketplace catalogs are maintained at the repository root; adding source alone does not publish a runnable package.

Use Go 1.27.3+ (the Go toolchain can fetch the pinned minimum automatically). Wails is pinned to v2.17.0 in go.mod. Frontend assets are plain embedded HTML/CSS/JS, so this repository's build path does not need an npm frontend build.

macOS GUI builds need Xcode command-line tools and CGO. Windows builds use Go and WebView2 on the user's machine. Linux GUI builds need GCC, pkg-config, GTK3 development files and WebKitGTK 4.1 development files; runtime systems need the corresponding libraries and a graphical session for setup. No platform needs an OS credential store.

```sh
go run ./scripts check
go vet ./src ./scripts
go run ./scripts build -target darwin/arm64
go run ./scripts build -target darwin/amd64
go run ./scripts build -target windows/amd64
go run ./scripts build -target windows/arm64
go run ./scripts build -target linux/amd64
go run ./scripts build -target linux/arm64
```

Build macOS/Linux GUI targets on the matching OS with the necessary native SDKs; ARM64 Linux builds need an ARM64 toolchain/host. Windows builds can be cross-compiled. Development-only `-headless` builds skip the webview for core compile checks; they are explicitly rejected by the packager and do not establish GUI support.

No persistent test suite is kept. Use disposable external fixtures and development builds to verify MCP initialization/tools, project/environment isolation, query rejection, TLS errors, cancellation, EOF shutdown, configuration migration, inline password persistence/redaction, UI CRUD/focus and dependency error messages. Never use real database credentials during build verification. Compile checks alone do not establish Windows/Linux host behavior.

## Verification performed

Disposable management fixtures verified credential/TLS/color preservation during renames, duplicate and invalid-input rejection without file changes, project/environment deletion scope, color persistence/redaction, prototype-like names and last-entry deletion. UI flow checks verified preset choices, draft preservation during renames/color changes, duplicate errors, delete cancellation and selection/empty-state recovery after deletion.

Disposable MySQL protocol fixtures verified setup connection testing with both saved and newly entered passwords, current unsaved database values, sanitized authentication failures, and byte-for-byte unchanged configuration after both successful and failed tests. Frontend checks covered unchanged/reverted Save state, password edits, testing progress/success/failure, Saved confirmation, duplicate-save prevention and preservation of edits after a save failure.

For the current inline-password implementation, disposable fixtures verified v1/v2 settings migration, plaintext password round trips, preserving unchanged/empty passwords, project/environment isolation, UI/status redaction, owner-only file permissions, missing-password errors and NUL rejection. MCP initialization, tool discovery, status, write rejection and EOF shutdown passed. Native macOS ARM64 and Windows x64 GUI builds, Linux ARM64 headless compilation and `go vet` passed. Windows/Linux runtime behavior remains unverified for this change.

Historical validation of the previous OS credential-store implementation on macOS ARM64 using disposable development builds and synthetic fixtures (credential-store results do not apply to the current inline-password implementation):

- MCP initialization, tool discovery, status, read-only MySQL queries, project/connection routing, blocked writes and side effects, TLS trust/hostname rejection, custom-CA success, cancellation and EOF cleanup.
- Actual Wails window, creating projects, adding/editing environments, native Keychain saves, empty-password reuse, simultaneous/repeated MCP setup calls sharing one window, and saved settings visibility.
- V1/v2 saved-data migration, original default-directory fallback, credential hash parity (including Unicode and literal escapes), project/environment isolation, and invalid configuration rejection.
- `go vet ./src ./scripts`, native source checks and whitespace checks.

Full GUI executables compile for macOS ARM64/x64 and Windows ARM64/x64. Linux ARM64/x64 core-only builds compile.

A full Linux ARM64 GUI executable also compiled and ran in a disposable Debian 13 container: GTK3/WebKitGTK 4.1 window startup/reuse under Xvfb, native Secret Service credentials, MCP/query safety checks, EOF cleanup, and a clear missing-desktop error all passed. WebKit sandboxing was disabled only in that disposable container to accommodate container restrictions; product code keeps its default sandbox behavior.

Windows WebView2/Credential Manager behavior, native Linux desktop/sandbox behavior, Linux x64 GUI compilation/runtime, macOS Intel runtime, and installation through the actual Claude/Codex plugin UI remain unverified. These are validation gaps, not a declaration of complete platform compatibility. No release was generated.

## Local desktop development

Build a small development wrapper once:

```sh
go run ./scripts dev
```

This builds the current machine's GUI executable at `build/<os>-<arch>/select` (or `.exe`) and creates `build/dev/select-dev.zip`. Both MCP manifests in this wrapper point to that executable's absolute path, instead of shipping a copy inside the plugin. Install the wrapper in Claude Cowork and ChatGPT/Codex desktop. When switching from `db-plugin`, add Select as a new plugin and disable the old plugin to avoid duplicate tools; subsequent wrapper updates use the `select` identity. Use local sessions running on this computer; the wrapper cannot launch this computer's executable from a cloud session or another machine. No credentials are included.

The local executable detects the repository's `src/ui/` directory from its location under `build/<os>-<arch>/` and reads the UI files directly. CSS edits apply automatically while the window is visible, without rebuilding or clearing form fields. Reopen setup to load HTML/JavaScript edits. `SELECT_UI_DIR` can explicitly select another absolute UI directory for isolated previews. Asset responses disable caching. Packaged executables outside the source build directory use their embedded UI.

For subsequent Go changes:

```sh
go run ./scripts build
```

Close any existing Select settings window and start a fresh MCP session, or fully quit and reopen the desktop app if it retains the old server. The next launch uses the rebuilt executable without uploading the wrapper again. Stop the clients before rebuilding on Windows, where a running executable cannot be replaced. This workflow does not hot-reload Go code in a running MCP server or a cached skill. Skill/manifest changes, moving the repository, or switching OS/CPU require regenerating and updating the wrapper. Normal release packages remain self-contained and continue bundling their executable and embedded UI.

Validation runs the development manifests from a copied temporary installation directory, checking MCP startup and tool discovery against the external build. Actual acceptance and launch through each desktop host must still be verified after the one-time installation.

References: [Claude local plugin components](https://claude.com/docs/plugins/build#check-what-each-app-loads), [OpenAI local plugin development](https://developers.openai.com/plugins/build/plugins#install-a-local-plugin-manually).

## Explicit release commands

Only run these when a release is requested:

```sh
go run ./scripts notices
go run ./scripts package -target darwin/arm64
```

The packager consumes `build/<os>-<arch>/select` (or `.exe`), validates its embedded Go target metadata, and writes platform-specific plugin ZIP, private marketplace ZIP, single-connection MCPB and SHA-256 checksums into `releases/`. While the project is WIP, filenames omit versions (for example, `select-darwin-arm64.zip`) and subsequent packages overwrite the same target's files. Existing version metadata is retained for client compatibility; packaging does not bump it. It does not build binaries or publish anything. `-binary /absolute/path` selects another existing GUI-enabled build. No top-level bin/ folder is shipped. Future signing/notarization requires the team's signing setup and explicit release authorization.
