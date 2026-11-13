# Distribution

The public catalog lives at `giunashvili/ai-plugins`. It lists Select only after a successful explicit local release creates `published/select/` and the resulting changes are committed and pushed. See the repository [README](../../../README.md) for the release command. No GitHub Actions or automatic commits are required.

## Install from the public repository

Each colleague adds `giunashvili/ai-plugins` as a marketplace source, then installs Select. In Claude, open **Plugins → Add → Add marketplace → Add from a repository**. This account-level installation is separate from centrally managed organization GitHub syncing; Claude currently requires a private/internal repository for that organization-sync flow.

ChatGPT/Codex uses `.agents/plugins/marketplace.json`; Claude uses `.claude-plugin/marketplace.json`. Both point at the same self-contained `published/select/` package. Its launcher selects macOS or Linux x64/ARM64; Windows resolves the shared command to the included `.exe` bootstrap, which starts the native x64/ARM64 executable. Windows ARM64 requires Windows 11 x64 emulation for the bootstrap. Only install one Select instance in a client to avoid duplicate tools.

The runtime must execute locally with a graphical desktop to open settings. Linux users can use a local MCP-capable client with a desktop session; publishing a Linux executable does not imply a Linux edition of Claude Cowork exists. Cloud and ordinary browser chat cannot open the native settings window.

## Dependencies and updates

Users need database network/VPN access and platform WebView libraries: WebView2 on Windows, GTK3 and WebKitGTK 4.1 on Linux. They do not install Go, Node, Docker or an OS credential-store service. Verify actual client installation, setup startup and a read-only query on each platform before claiming it tested. Signing and notarization are separate from building the binaries.

Update via the client's marketplace controls and start a fresh session when required. Project/environment settings and plaintext passwords remain in each user's local configuration directory, outside the package. Never distribute another person's settings, credentials, private keys or private database hosts.

The lower-level per-target packager still generates `select-OS-ARCH.zip`, a per-target marketplace ZIP and the separate `select-connection-OS-ARCH.mcpb`. These filenames omit versions while WIP. The MCPB is for ordinary Claude Desktop single-connection chat and does not expose the project manager.

References: [Claude installation](https://support.claude.com/en/articles/13837440-use-plugins-in-claude), [organization distribution](https://support.claude.com/en/articles/13837433-manage-plugins-for-your-organization), [OpenAI marketplace import](https://learn.chatgpt.com/docs/enterprise/plugin-management), [Wails prerequisites](https://v2.wails.io/docs/gettingstarted/installation/).
