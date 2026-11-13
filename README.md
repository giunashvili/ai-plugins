# AI Plugins

Private marketplace catalog for shared AI plugins. No plugins are published yet.

## Structure

```text
ai-plugins/
├── .agents/plugins/marketplace.json  # ChatGPT / Codex catalog
├── .claude-plugin/marketplace.json  # Claude catalog
├── plugins/                        # Ready-to-install plugin folders
│   └── .gitkeep
├── .gitignore
└── README.md
```

## Adding a plugin

1. Build and test the plugin in its source repository.
2. Place its distribution files in `plugins/<plugin-name>/`, including the required manifests, skills and compiled executable when applicable.
3. Add an entry to each marketplace catalog using that host's format. Paths are relative to this repository's root.
4. Give the plugin its own version and verify its supported operating systems and processors.
5. Merge the reviewed publishing pull request into `main`, then verify marketplace synchronization.

Keep development builds in the source repository. Publish tested updates here; use Git tags to record released catalog revisions.

Never include local connection settings, passwords, private keys or machine-specific development wrappers.
