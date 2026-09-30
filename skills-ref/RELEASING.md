# Releasing skills-ref

Releases are cut from `main` with
[release-please](https://github.com/googleapis/release-please) and built with
GoReleaser in
[`skills-ref-release.yml`](../.github/workflows/skills-ref-release.yml).
Release URLs and the [installer prompt](INSTALL_PROMPT.md) point to
`skills-repos/agentskills`.

## How a release happens

1. Commits that touch `skills-ref/` land on `main` with
   [Conventional Commit](https://www.conventionalcommits.org/) messages.
   Before 1.0.0, `feat:` bumps the minor version, `fix:` bumps the patch, and
   `feat!:` or a `BREAKING CHANGE:` footer bumps the minor version.
2. On each push to `main`, the `release-please` job opens or updates a release
   PR. The PR updates `skills-ref/CHANGELOG.md` and
   [`.release-please-manifest.json`](../.release-please-manifest.json).
3. Merging the release PR creates the `skills-ref/vX.Y.Z` tag and a draft
   GitHub release. This tag format is also the one `go install` uses for a
   module in a subdirectory.
4. The `publish` job then runs on that tag:
   - runs the tests;
   - builds the archives with GoReleaser;
   - checks that the binary reports the version;
   - checks that `checksums.txt` covers every installable asset and that
     `install.sh` succeeds against the staged assets;
   - uploads the assets;
   - publishes the release and marks it as latest.

   The release stays a draft until every asset is uploaded, so
   `releases/latest` never points at a release without binaries. Once the
   release is published, the
   [install canary](../.github/workflows/skills-ref-install-canary.yml) runs
   the released installer on Linux, macOS and Windows after each
   `skills-ref release` run, and again every week; it fails loudly if a
   `releases/latest` URL is missing or a checksum does not match. Releases
   are published with `GITHUB_TOKEN`, whose events trigger no workflows, so
   the canary chains off `workflow_run` rather than `release: published`.

## Release assets

| Asset | Contents |
| --- | --- |
| `skills-ref_darwin_amd64.tar.gz` | macOS, Intel |
| `skills-ref_darwin_arm64.tar.gz` | macOS, Apple silicon |
| `skills-ref_linux_amd64.tar.gz`, `skills-ref_linux_arm64.tar.gz` | Linux |
| `skills-ref_windows_amd64.zip`, `skills-ref_windows_arm64.zip` | Windows |
| `SKILL.md` | The installable skill from [`example/skills-ref`](example/skills-ref/SKILL.md) |
| `install.sh`, `install.ps1` | Installer scripts referenced by the [installer prompt](INSTALL_PROMPT.md), from [`install/`](install/) |
| `checksums.txt` | SHA-256 of every archive, `SKILL.md` and both installer scripts |
| `<archive>.sbom.json` | SPDX SBOM per archive |

Names carry no version, so
`https://github.com/skills-repos/agentskills/releases/latest/download/<asset>`
always serves the newest release. Every archive holds the binary, `README.md`
and `LICENSE`. Builds are reproducible: file times come from the commit and
`-trimpath` removes local paths.

The macOS binaries are ad hoc signed by the Go linker, which Apple silicon
requires, but they are not notarized. Files downloaded with `curl` are not
quarantined and run directly. Browser downloads need
`xattr -d com.apple.quarantine skills-ref`.

## One-time repository setup

1. In Settings > Actions > General, set Workflow permissions to "Read and
   write" and enable "Allow GitHub Actions to create and approve pull
   requests".
2. [`release-please-config.json`](../release-please-config.json) has no
   `bootstrap-sha`, so release-please reads the whole history of `main`. To
   force a specific version, add a `Release-As: X.Y.Z` footer to a commit
   that touches `skills-ref/`.
3. Optional: add a fine-grained token or GitHub App token as a `token` input
   to the release-please step. With the default `GITHUB_TOKEN`, release PRs do
   not trigger the `skills-ref` CI workflow.

## Rollout plan

1. Commit the Go switch on a branch with Conventional Commit messages, for
   example `feat(skills-ref)!: replace Python reference library with Go CLI`.
2. Push the branch to the fork. Confirm that `skills-ref.yml` passes on
   ubuntu, macos and windows.
3. Merge to `main`. Check that release-please opens a PR titled
   `chore(main): release skills-ref 0.1.0`.
4. Merge the release PR. Check that the `publish` job succeeds and that the
   release is published as latest with 16 assets: 6 archives, 6 SBOMs,
   `SKILL.md`, `checksums.txt`, `install.sh` and `install.ps1`.
5. Verify from a clean machine or container for each OS family: paste the
   prompt from [`INSTALL_PROMPT.md`](INSTALL_PROMPT.md) (or the identical one
   in the repository README) into an agent on macOS, then on Linux and
   Windows. Confirm `skills-ref --version` and
   `skills-ref validate ~/.agents/skills/skills-ref` succeed; the CLI lives in
   `~/.local/bin`, which may need to be added to `PATH` first.

## Fixing a bad release

- If `publish` fails, the release stays a draft. Fix the cause on `main`, then
  run the workflow manually from Actions > skills-ref release > Run workflow
  with the tag, for example `skills-ref/v1.0.0`. Do not use "Re-run jobs":
  a re-run uses the workflow file from the original commit and would repeat
  the failure. `gh release upload --clobber` replaces partial uploads.
- To withdraw a published release, delete it and its tag
  (`gh release delete skills-ref/vX.Y.Z --cleanup-tag`), then ship a fix in a
  new patch release. Do not reuse a version number.

## Local checks

```sh
make snapshot   # the same archives in dist/, nothing published
```

To try the installer against the snapshot, stage the extra assets and serve
`dist/` locally, then point the installer at it:

```sh
cp example/skills-ref/SKILL.md install/install.sh install/install.ps1 dist/
python3 -m http.server -d dist 8000 &
SKILLS_REF_BASE_URL=http://localhost:8000 sh install/install.sh
```

## Later

- Notarize the macOS binaries (requires an Apple Developer ID), or publish a
  Homebrew tap.
- Pin third-party actions to commit SHAs.
- Add build provenance with `actions/attest-build-provenance` once the
  repository is public. GitHub does not support attestations for private
  repositories owned by a personal account.
