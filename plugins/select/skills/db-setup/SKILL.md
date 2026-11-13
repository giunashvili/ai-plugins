---
name: db-setup
description: Configure or troubleshoot MySQL projects and environments in Select's native desktop settings window.
---

# Set up Select

1. Call `db_status()` to list configured projects/environments. This does not verify connectivity.
2. Call `db_setup()` when configuration is needed. It opens or focuses the desktop settings window and returns `{opened: true}` once ready. There is no browser URL.
3. Let the user create projects/environments and enter credentials in that window. Never request passwords in chat, inspect entered values, fill password fields, or pass credentials through MCP tools.
4. When the user finishes, call `db_status()` again. For a requested check, call `mysql_query` with explicit `project`, `environment`, and `sql: "SELECT 1 AS db_connection_check"`. Only report connectivity after success.

Users need no Go or Node runtime or OS credential store. They do need the native executable for their local execution OS/CPU, database network/VPN access and the required WebView libraries. Passwords are stored as plaintext in the local profiles.json with restrictive file permissions. Never read that file to display passwords or copy it into shared artifacts. Older connections without an inline password need the user to save it once in setup. Report missing graphical sessions or dependencies clearly; do not switch to a remote service.

The separate ordinary Claude Desktop MCPB is a single-connection alternative; its query tool accepts only `sql`. Read README.md and docs/maintenance.md for prerequisites. Never build a release or register duplicate MCP servers as part of user setup.
