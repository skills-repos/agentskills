# Agent Skills

[![Discord](https://img.shields.io/badge/Discord-Join-5865F2?logo=discord&logoColor=white)](https://discord.gg/MKPE9g8aUy)

A standardized way to give AI agents new capabilities and expertise.

## Install the reference CLI with your agent

The [`skills-ref`](skills-ref/) reference CLI validates skills and prepares
skill metadata for agent prompts. To install it without curl-to-bash, paste
the prompt below into a coding agent that can run shell commands. The agent
detects your platform (macOS, Linux, or Windows), downloads the installer from
the [latest release](https://github.com/skills-repos/agentskills/releases/latest),
verifies its checksum, and shows you the script before running it. The
checksums come from the repository's release pipeline, so every download is
tied to the tagged release commit.

```text
Install the skills-ref Agent Skill and CLI from the latest GitHub release of https://github.com/skills-repos/agentskills. Download only from that repository's release page.

1. Detect my operating system (macOS, Linux, or Windows) and CPU architecture, and tell me what you detected.
2. Download the installer for my OS (install.sh, or install.ps1 on Windows) and checksums.txt from https://github.com/skills-repos/agentskills/releases/latest/download/ and verify the installer's SHA-256 against checksums.txt. If it does not match, stop and tell me.
3. Show me the installer script, then run it. It installs the CLI into ~/.local/bin and the skill into ~/.agents/skills/skills-ref, verifying every download against checksums.txt itself and stopping on a mismatch.
4. If the installer reports that its bin directory is not on my PATH, offer to add it to my shell profile.
5. Run `skills-ref --version` and `skills-ref validate ~/.agents/skills/skills-ref`, then report the detected platform, the version, and the validation result.
6. If you load skills from a different directory than ~/.agents/skills, tell me which one and ask before copying or linking the skill there.
```

For manual installation and usage, see [`skills-ref/README.md`](skills-ref/README.md).

## What are Agent Skills?

Agent Skills are a lightweight, open format for extending AI agent capabilities with specialized knowledge and workflows.

At its core, a skill is a folder containing a `SKILL.md` file. This file includes metadata (`name` and `description`, at minimum) and instructions that tell an agent how to perform a specific task. Skills can also bundle scripts, reference materials, templates, and other resources.

```
my-skill/
├── SKILL.md          # Required: metadata + instructions
├── scripts/          # Optional: executable code
├── references/       # Optional: documentation
├── assets/           # Optional: templates, resources
└── ...               # Any additional files or directories
```

## Why Agent Skills?

Agents are increasingly capable, but often don't have the context they need to do real work reliably. Skills solve this by packaging procedural knowledge and company-, team-, and user-specific context into portable, version-controlled folders that agents load on demand. This gives agents:

- **Domain expertise**: Capture specialized knowledge — from legal review processes to data analysis pipelines to presentation formatting — as reusable instructions and resources.
- **Repeatable workflows**: Turn multi-step tasks into consistent, auditable procedures.
- **Cross-product reuse**: Build a skill once and use it across any skills-compatible agent.

## How do Agent Skills work?

Agents load skills through **progressive disclosure**, in three stages:

1. **Discovery**: At startup, agents load only the name and description of each available skill, just enough to know when it might be relevant.

2. **Activation**: When a task matches a skill's description, the agent reads the full `SKILL.md` instructions into context.

3. **Execution**: The agent follows the instructions, optionally executing bundled code or loading referenced files as needed.

Full instructions load only when a task calls for them, so agents can keep many skills on hand with only a small context footprint.

## Where can I use Agent Skills?

Agent Skills are supported by a large number of AI tools and agentic clients — see the [Client Showcase](https://agentskills.io/clients) to explore some of them!

## Getting started

- **[Documentation](https://agentskills.io)** — Guides and tutorials
- **[Specification](https://agentskills.io/specification)** — Format details
- **[Example Skills](https://github.com/anthropics/skills)** — See what's possible
- **[Discord](https://discord.gg/MKPE9g8aUy)** — Share what you're building!

## Open development

The Agent Skills format was originally developed by [Anthropic](https://www.anthropic.com/), released as an open standard, and has been adopted by a growing number of agent products. The standard is open to contributions from the broader ecosystem — see [`CONTRIBUTING.md`](CONTRIBUTING.md) for how to get involved.

## License

Code in this repository is licensed under [Apache 2.0](LICENSE). Documentation is licensed under [CC-BY-4.0](https://creativecommons.org/licenses/by/4.0/). See individual directories for details.
