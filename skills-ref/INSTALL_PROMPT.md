# Install skills-ref with your agent

Copy the prompt below into a coding agent that can run shell commands. It
installs the `skills-ref` CLI and skill from the latest release of this
repository: the CLI goes to `~/.local/bin` and the skill to
`~/.agents/skills/skills-ref`. The installer verifies every download against
`checksums.txt`, which the release pipeline generates for the tagged commit.

```text
Install the skills-ref Agent Skill and CLI from the latest GitHub release of https://github.com/skills-repos/agentskills. Download only from that repository's release page.

1. Detect my operating system (macOS, Linux, or Windows) and CPU architecture, and tell me what you detected.
2. Download the installer for my OS (install.sh, or install.ps1 on Windows) and checksums.txt from https://github.com/skills-repos/agentskills/releases/latest/download/ and verify the installer's SHA-256 against checksums.txt. If it does not match, stop and tell me.
3. Show me the installer script, then run it. It installs the CLI into ~/.local/bin and the skill into ~/.agents/skills/skills-ref, verifying every download against checksums.txt itself and stopping on a mismatch.
4. If the installer reports that its bin directory is not on my PATH, offer to add it to my shell profile.
5. Run `skills-ref --version` and `skills-ref validate ~/.agents/skills/skills-ref`, then report the detected platform, the version, and the validation result.
6. If you load skills from a different directory than ~/.agents/skills, tell me which one and ask before copying or linking the skill there.
```

To update later, paste the same prompt again; it replaces the CLI and skill
with the latest release.
