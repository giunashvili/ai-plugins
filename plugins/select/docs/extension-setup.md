# Claude Desktop single-connection extension

The platform-specific `select-connection` MCPB is a separate alternative for ordinary Claude Desktop chat. It uses the same native executable in `desktop` mode and manages one connection through Claude's extension settings. It does not expose the project manager.

Install the extension matching the OS/CPU on which Claude runs. Enter project/environment labels, host, port, database, username, password and TLS settings. The host stores the sensitive password field and supplies `DB_MCPB_*` environment values only to the local process. Its `mysql_query` tool accepts `sql` only. Changing extension settings replaces that connection.

No Go or Node installation is needed. The full executable's OS webview libraries must still be available. Use a read-only MySQL account and verify connectivity before relying on query results. This store is separate from the main plugin's project manager.
