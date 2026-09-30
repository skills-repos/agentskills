# Install the skills-ref CLI and Agent Skill from the latest release of
# https://github.com/skills-repos/agentskills
#
# Downloads the archive for this platform, SKILL.md and checksums.txt,
# verifies SHA-256 checksums, then installs the CLI into $env:BIN_DIR
# (default: ~\.local\bin) and the skill into $env:SKILL_DIR
# (default: ~\.agents\skills\skills-ref).
#
# Overrides for testing: SKILLS_REF_BASE_URL, BIN_DIR, SKILL_DIR.
$ErrorActionPreference = 'Stop'

$base = if ($env:SKILLS_REF_BASE_URL) { $env:SKILLS_REF_BASE_URL } else { 'https://github.com/skills-repos/agentskills/releases/latest/download' }
$binDir = if ($env:BIN_DIR) { $env:BIN_DIR } else { "$HOME\.local\bin" }
$skillDir = if ($env:SKILL_DIR) { $env:SKILL_DIR } else { "$HOME\.agents\skills\skills-ref" }

$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }
$asset = "skills-ref_windows_$arch.zip"

$tmp = New-Item -ItemType Directory -Path (Join-Path ([IO.Path]::GetTempPath()) ([guid]::NewGuid()))
try {
    Write-Host "Downloading $asset, SKILL.md and checksums.txt for windows/$arch"
    Invoke-WebRequest "$base/$asset" -OutFile "$tmp\$asset"
    Invoke-WebRequest "$base/SKILL.md" -OutFile "$tmp\SKILL.md"
    Invoke-WebRequest "$base/checksums.txt" -OutFile "$tmp\checksums.txt"

    foreach ($f in @($asset, 'SKILL.md')) {
        $line = Select-String -Path "$tmp\checksums.txt" -Pattern ('  ' + [regex]::Escape($f) + '$')
        if (-not $line) { throw "no checksum for $f in checksums.txt" }
        $want = $line.Line.Split(' ')[0]
        $got = (Get-FileHash "$tmp\$f" -Algorithm SHA256).Hash.ToLower()
        if ($got -ne $want) { throw "checksum mismatch for $f" }
    }

    New-Item -ItemType Directory -Force -Path $binDir, $skillDir | Out-Null
    Expand-Archive "$tmp\$asset" -DestinationPath $tmp
    Move-Item -Force "$tmp\skills-ref.exe" "$binDir\skills-ref.exe"
    Move-Item -Force "$tmp\SKILL.md" "$skillDir\SKILL.md"
}
finally {
    Remove-Item -Recurse -Force $tmp
}

Write-Host "Installed CLI:   $binDir\skills-ref.exe"
Write-Host "Installed skill: $skillDir\SKILL.md"
& "$binDir\skills-ref.exe" --version

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (($userPath -split ';') -notcontains $binDir) {
    Write-Host "NOTE: $binDir is not on your user PATH. Add it with:"
    Write-Host "  [Environment]::SetEnvironmentVariable('Path', [Environment]::GetEnvironmentVariable('Path','User') + ';$binDir', 'User')"
}
