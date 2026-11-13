# AI Plugins

Public source repository and marketplace catalog for shared AI plugins.

## Structure

```text
ai-plugins/
├── .agents/plugins/marketplace.json  # ChatGPT / Codex catalog
├── .claude-plugin/marketplace.json  # Claude catalog
├── plugins/select/                 # Select source, skills, assets and tooling
├── scripts/release.go              # Explicit local release command
└── published/select/               # Installable package, created on release
```

## Development

```sh
cd plugins/select
go run ./scripts check
go run ./scripts dev
```

The machine-specific development wrapper and builds stay under `plugins/select/build/` and are Git-ignored. See [Select maintenance](plugins/select/docs/maintenance.md).

## Make a release

Run from the repository root only when a release is requested:

```sh
go run ./scripts/release.go select
```

The release coordinator is a Mac with Xcode command-line tools, Go and running Docker Desktop. It builds macOS and Windows x64/ARM64 binaries locally, and Linux x64/ARM64 GUI binaries in disposable Debian containers. Docker is a build dependency; users do not need Docker, Go or Node.

The command validates all six production binaries, assembles `published/select/`, writes the unversioned `releases/select.zip` and checksum file, and updates both catalogs to `./published/select`. An OS/CPU launcher selects the matching native binary; no per-platform marketplace entries are needed. `published/` is tracked, while ZIP archives are ignored.

To inspect the plan without building or publishing files:

```sh
go run ./scripts/release.go select -check
```

For builds supplied by other machines, `-binaries /absolute/path` explicitly reuses a directory containing `darwin-amd64/select`, `darwin-arm64/select`, `linux-amd64/select`, `linux-arm64/select`, `windows-amd64/select.exe` and `windows-arm64/select.exe`. The script validates target metadata and production tags; the publisher must verify those supplied binaries match the current source.

Before subsequent releases, increment the same internal version in the three Select manifests and `plugins/select/src/mcp.go`. Archive filenames remain unversioned. The command refuses to replace an existing release with the same version.

The script never runs Git commands, creates commits or pushes/uploads anything. Review the generated changes, then commit and push only when explicitly requested. There are no GitHub Actions requirements.

Both catalogs stay empty until the first successful local release. Creating this script has not published a release.

## Installation

After the first release is committed and pushed, add `giunashvili/ai-plugins` as a repository marketplace in the client, then install Select. In Claude, use **Plugins → Add → Add marketplace → Add from a repository**. Each person adds the source to their account; this is separate from an organization administrator configuring a centrally managed GitHub sync. Claude's organization GitHub sync currently requires a private/internal repository.

Select needs local MCP execution with a graphical desktop for settings, database network access and native WebView libraries. Linux requires GTK3 and WebKitGTK 4.1; Windows requires WebView2. The x64 Windows launcher uses Windows 11's x64 emulation on ARM64 to start the native ARM64 runtime. Cross-platform host installation and GUI behavior must be verified before declaring a release tested.

Database connections and passwords remain in each user's local `profiles.json`, outside the plugin. Never include local settings, passwords or private keys in this repository or a release.

Sources: [Claude marketplace installation](https://support.claude.com/en/articles/13837440-use-plugins-in-claude), [Claude organization distribution](https://support.claude.com/en/articles/13837433-manage-plugins-for-your-organization), [OpenAI marketplace import](https://learn.chatgpt.com/docs/enterprise/plugin-management).
