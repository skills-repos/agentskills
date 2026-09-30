---
name: skills-ref
description: Validate Agent Skills folders and prepare skill metadata for an agent prompt. Use when checking a SKILL.md file, building an available-skills list, or installing the skills-ref CLI.
---

# Check or list skills

## Find the CLI

Use `skills-ref` from `PATH`. If it is not installed, install it as described
below.

## Check a skill

Run `skills-ref validate --format json path/to/skill`. Inspect
`results[].problems` for stable `code` and a human-readable `message`; exit
code 1 means one or more skills failed validation, and exit code 2 means a
usage error or a missing path. For multiple skills, pass multiple paths or
`--recursive`.

To display properties, run `skills-ref read-properties path/to/skill`. To make
an Anthropic-style `<available_skills>` block, run
`skills-ref to-prompt path/to/skill-a path/to/skill-b`. A client may choose a
different prompt format. The binary only reads files; do not execute a skill's
scripts merely to validate it.

## Install the CLI

Download only from the latest release of
`https://github.com/skills-repos/agentskills`. Get `install.sh` (macOS and
Linux) or `install.ps1` (Windows PowerShell) together with `checksums.txt`,
verify the script's SHA-256 against `checksums.txt`, then run it. Stop on a
checksum mismatch. The installer detects the platform, verifies every
download, puts the CLI in `~/.local/bin` and this skill in
`~/.agents/skills/skills-ref`, and prints a line to add to the shell profile
if the bin directory is not on `PATH`.
