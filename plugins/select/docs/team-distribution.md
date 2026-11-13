# Private distribution

No releases are generated until explicitly requested. WIP packages are specific to one OS and processor and omit versions: `select-OS-ARCH.zip`, `select-marketplace-OS-ARCH.zip`, and `select-connection-OS-ARCH.mcpb`. Go uses `amd64` for x64 and `arm64` for ARM64.

## Claude

For a personal installation, use Customize → Plugins → Add → Upload plugin, select the ZIP matching the local execution machine, and enable Select. Start a new Claude session that executes MCP on your local desktop, then ask to open Select setup. The desktop settings window collects connection credentials.

For a team, an owner can upload the matching plugin ZIP through Organization settings → Plugins & skills, or host the supplied marketplace in a private repository. Keep platform variants in separate marketplaces with clear platform labels; uploading another platform under the same name in the same marketplace would replace its executable. Users install the variant matching their machine. Plugin identity remains select.

## Codex

Use the matching platform's private marketplace source. The root portable manifest launches the included native executable over stdio. Each platform variant contains the same tools and skills, without a Node requirement. Keep platform-specific marketplaces clearly labeled. For WSL, choose the Linux variant corresponding to the WSL execution environment.

The current checked-in manifests describe the Unix executable path. The explicit packager generates Windows manifests ending in `.exe`. Ordinary Claude Desktop chat uses the separate MCPB single-connection extension, not the project manager.

## Prerequisites and updates

The MCP process must run in the same desktop session as the settings window. Sandboxed/cloud executors without a desktop cannot open it. Teammates need the platform's native webview libraries and database network access. They do not install Go, Node or an OS credential-store service. Update through the same platform's marketplace and start a new session. Project/environment settings and plaintext passwords stay in the user's configuration folder, outside the package. After updating from an OS-store build, save each missing password once in setup.

Do not distribute another person's settings, credential files, private keys or database hosts. No automatic publishing or account installation is performed by developer build commands. Signing/notarization and actual client/OS installation must be verified before a team release.

References: [Claude plugins](https://claude.com/docs/plugins/overview), [organization distribution](https://support.claude.com/en/articles/13837433-manage-plugins-for-your-organization), [native Wails prerequisites](https://v2.wails.io/docs/gettingstarted/installation/).
