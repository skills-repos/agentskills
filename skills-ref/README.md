# skills-ref

Reference command-line tool for Agent Skills, written in Go.

> [!IMPORTANT]
> This tool is intended for demonstration purposes only. It is not meant to be used in production.

## Installation

With Go 1.26 or later:

```sh
go install github.com/skills-repos/agentskills/skills-ref/cmd/skills-ref@latest
```

To install the CLI and skill with a coding agent, paste the prompt in
[`INSTALL_PROMPT.md`](INSTALL_PROMPT.md). The agent runs the verified
`install.sh` or `install.ps1` from the release, which puts the CLI in
`~/.local/bin` and the skill in `~/.agents/skills/skills-ref`.

Or download the archive for your platform and `checksums.txt` from the
[latest release](https://github.com/skills-repos/agentskills/releases/latest),
and verify the checksum before use:

```sh
sha256sum --ignore-missing -c checksums.txt   # macOS: shasum -a 256 --ignore-missing -c
tar -xzf skills-ref_<os>_<arch>.tar.gz skills-ref
```

`<os>` is `darwin` (macOS), `linux` or `windows`, and `<arch>` is `amd64` or
`arm64`. Windows archives are `.zip`. Each archive holds the `skills-ref`
binary, this README and the license, and has an SPDX SBOM
(`<archive>.sbom.json`). The release also carries the installable
[`SKILL.md`](example/skills-ref/SKILL.md).

## Usage

```sh
skills-ref validate path/to/skill
skills-ref validate --format json path/to/skill
skills-ref validate --recursive skills/
skills-ref read-properties path/to/skill
skills-ref to-prompt path/to/skill-a path/to/skill-b
skills-ref help [command]
skills-ref --version
```

Each path is a skill directory or its `SKILL.md` file. Options may come before
or after paths; arguments after `--` are always paths. Every command exits 0 on
success, 1 for an invalid or unreadable skill, and 2 for a usage error or a
path that does not exist.

- `validate --format json` reports stable problem codes and follows
  [`conformance/schema/validate.json`](conformance/schema/validate.json).
- `validate --recursive` checks every skill below each path. It follows a
  symlinked path but not symlinks inside it, reports each skill once, and
  fails if a path contains no skills.
- `read-properties` prints the frontmatter as JSON.
- `to-prompt --skip-invalid` leaves out skills that fail validation and warns
  on stderr.

## Agent prompt integration

`to-prompt` prints the suggested `<available_skills>` XML block for an agent's
system prompt. This format is recommended for Anthropic's models, but clients
may format it differently for the model they use.

```xml
<available_skills>
<skill>
<name>
my-skill
</name>
<description>
What this skill does and when to use it
</description>
<location>
/path/to/my-skill/SKILL.md
</location>
</skill>
</available_skills>
```

The `<location>` element tells the agent where to find the full skill
instructions.

## Development

```sh
make build      # ./skills-ref
make test       # unit and conformance tests with the race detector
make lint       # gofmt, go vet, staticcheck
make snapshot   # local release archives in dist/; nothing is published
```

[`conformance/`](conformance/) holds end-to-end CLI cases that `make test` runs
against a freshly built binary.

## Releases

Releases are cut from `main` by merging a release PR. See
[`RELEASING.md`](RELEASING.md).

## License

Apache 2.0
