# AI Plugins

Private repository for shared AI plugin source and marketplace catalogs.

## Structure

```text
ai-plugins/
├── .agents/plugins/marketplace.json  # ChatGPT / Codex catalog
├── .claude-plugin/marketplace.json  # Claude catalog
├── plugins/
│   └── select/                     # Select source, skills and build tooling
│       ├── .claude-plugin/
│       ├── .codex-plugin/
│       ├── src/                    # Go backend and HTML/CSS/JavaScript UI
│       ├── scripts/                # Build and packaging commands
│       ├── skills/
│       ├── docs/
│       ├── go.mod
│       ├── plugin.json
│       └── README.md
├── .gitignore
└── README.md
```

## Select development

Run commands from `plugins/select/`:

```sh
cd plugins/select
go run ./scripts check
go run ./scripts dev
```

Builds and the machine-specific development wrapper go into `plugins/select/build/`. Release packaging is explicit; see [Select maintenance](plugins/select/docs/maintenance.md).

Each plugin owns its source, manifests, skills, dependencies and build tooling. Generated builds and release archives are Git-ignored.

## Publishing

The marketplace catalogs are currently empty. Select source has moved here, but it has not been published to either marketplace.

Before registering a plugin, build and test its supported platforms and provide the compiled executable required by its MCP manifests. Merely committing Go source does not make Select installable. Do not publish machine-specific development wrappers.

Add each ready-to-install plugin to both catalogs using the host-specific format. Paths are relative to this repository's root. Merge reviewed updates into `main` with the plugin's version bump; Git tags can record released catalog revisions.

Never include local connection settings, passwords or private keys.
