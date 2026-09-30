#!/bin/sh
# Fail if the installer prompt in README.md differs from the canonical one in
# skills-ref/INSTALL_PROMPT.md. Both files carry the same fenced ```text block.
set -eu

cd "$(dirname "$0")/../.."
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

extract() {
  awk '
    /^```text$/ { inblock = 1; next }
    inblock && /^```$/ { exit }
    inblock { print }
  ' "$1"
}

extract README.md > "$tmp/readme"
extract skills-ref/INSTALL_PROMPT.md > "$tmp/install_prompt"

if ! grep -q . "$tmp/install_prompt"; then
  echo "No \`\`\`text block found in skills-ref/INSTALL_PROMPT.md" >&2
  exit 1
fi
if ! diff -u "$tmp/install_prompt" "$tmp/readme"; then
  echo "README.md and skills-ref/INSTALL_PROMPT.md carry different installer prompts." >&2
  exit 1
fi
