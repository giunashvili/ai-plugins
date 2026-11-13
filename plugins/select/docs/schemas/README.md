# Validation schema provenance

`mcpb-manifest-v0.3.schema.json` is the unmodified MCPB 0.3 JSON Schema downloaded on 2026-11-11 from:

https://raw.githubusercontent.com/modelcontextprotocol/mcpb/main/schemas/mcpb-manifest-v0.3.schema.json

The repository's license at that time is preserved in `MCPB-LICENSE`, including its licensing-transition notice. Source: https://github.com/modelcontextprotocol/mcpb.

Package builds validate both generated extension manifests against this schema using Ajv. Additional local checks verify the entrypoint, bundled runtime, profile selection, user-config references, and sensitive password field. The schema is build tooling and is not needed by the extension runtime.

## Portable plugin schemas

`agent-plugin-v1.schema.json` and `agent-mcp-v1.schema.json` are unmodified copies of the Agent Plugins 1.0.0 JSON schemas, downloaded on 2026-11-11. They are validated with Ajv's JSON Schema 2020-12 support.

- https://agent-plugins.org/schemas/1.0.0/plugin.schema.json
- https://agent-plugins.org/schemas/1.0.0/mcp.schema.json

These schemas validate the portable files. Client-specific OpenAI extension fields and Claude user configuration also receive semantic checks; Claude's own CLI validates its native plugin manifest.
