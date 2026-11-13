---
name: setup
description: Configure Select's MySQL connections, projects and environments, or troubleshoot connection failures.
---

# Setup

- Call `db_setup` to open/focus settings. `{opened: true}` means the window is ready, not that a connection works.
- Manage projects, environments, colors and credentials in that window. The user enters passwords there, never in chat.
- **Test connection** checks unsaved values; **Save** persists them. A blank password preserves an existing saved password.
- `db_status` lists saved names, not connectivity. For a requested connection check, call `mysql_query` with the selected project/environment and `SELECT 1`.
- Follow the returned diagnostic. Do not automatically disable TLS. For desktop/WebView failures, consult [requirements](../../README.md).
