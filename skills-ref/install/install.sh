#!/bin/sh
# Install the skills-ref CLI and Agent Skill from the latest release of
# https://github.com/skills-repos/agentskills
#
# Downloads the archive for this platform, SKILL.md and checksums.txt,
# verifies SHA-256 checksums, then installs the CLI into $BIN_DIR
# (default: ~/.local/bin) and the skill into $SKILL_DIR
# (default: ~/.agents/skills/skills-ref).
#
# Overrides for testing: SKILLS_REF_BASE_URL, BIN_DIR, SKILL_DIR.
set -eu

base="${SKILLS_REF_BASE_URL:-https://github.com/skills-repos/agentskills/releases/latest/download}"
bin_dir="${BIN_DIR:-$HOME/.local/bin}"
skill_dir="${SKILL_DIR:-$HOME/.agents/skills/skills-ref}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  darwin | linux) ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) echo "unsupported CPU: $(uname -m)" >&2; exit 1 ;;
esac

asset="skills-ref_${os}_${arch}.tar.gz"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
cd "$tmp"

echo "Downloading $asset, SKILL.md and checksums.txt for $os/$arch"
curl -fsSLO "$base/$asset"
curl -fsSLO "$base/SKILL.md"
curl -fsSLO "$base/checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then sum="sha256sum"; else sum="shasum -a 256"; fi
for f in "$asset" SKILL.md; do
  line="$(grep "  $f\$" checksums.txt)" || { echo "no checksum for $f in checksums.txt" >&2; exit 1; }
  printf '%s\n' "$line" | $sum -c - >/dev/null
done

mkdir -p "$bin_dir" "$skill_dir"
tar -xzf "$asset" skills-ref
mv skills-ref "$bin_dir/skills-ref"
mv SKILL.md "$skill_dir/SKILL.md"

echo "Installed CLI:   $bin_dir/skills-ref"
echo "Installed skill: $skill_dir/SKILL.md"
"$bin_dir/skills-ref" --version

case ":$PATH:" in
  *":$bin_dir:"*) ;;
  *)
    echo "NOTE: $bin_dir is not on your PATH. Add this line to your shell profile:" >&2
    echo "  export PATH=\"$bin_dir:\$PATH\"" >&2
    ;;
esac
