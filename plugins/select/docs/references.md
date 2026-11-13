# Implementation references

- [Official Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk)
- [Go MySQL driver](https://github.com/go-sql-driver/mysql)
- [TiDB MySQL parser](https://github.com/pingcap/tidb/tree/release-8.5/pkg/parser)
- [Wails v2 platform prerequisites](https://v2.wails.io/docs/gettingstarted/installation/)
- [Claude plugin manifests](https://code.claude.com/docs/en/plugins-reference)
- [Claude hosted upload requirements](https://platform.claude.com/docs/en/manage-claude/plugins-api)
- [Portable Agent Plugins](https://agent-plugins.org/)
- [MCPB binary server format](https://github.com/modelcontextprotocol/mcpb)

Pinned dependency versions are in go.mod/go.sum. Vendored JSON schemas under docs/schemas are retained as format references; native source checks verify project-specific manifest wiring and archive paths.
