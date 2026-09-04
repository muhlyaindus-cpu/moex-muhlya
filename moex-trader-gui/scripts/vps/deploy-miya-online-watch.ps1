param(
    [string]$HostName = "YOUR_VPS_HOST",
    [string]$UserName = "traderbot",
    [string]$AppDir = "/opt/trading/app/moex-trader-gui"
)

$ErrorActionPreference = "Stop"

function Require-File {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        throw "File not found: $Path"
    }
}

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$localScript = Join-Path $repoRoot "scripts\vps\miya-online-watch.sh"
$localService = Join-Path $repoRoot "scripts\vps\systemd\miya-online-watch.service"
$localTimer = Join-Path $repoRoot "scripts\vps\systemd\miya-online-watch.timer"
$localInstall = Join-Path $repoRoot "scripts\vps\install-miya-online-watch-systemd-remote.sh"

Require-File -Path $localScript
Require-File -Path $localService
Require-File -Path $localTimer
Require-File -Path $localInstall

$target = "$UserName@$HostName"

Write-Host "Uploading Miya online monitor files to $target ..."
scp $localScript "$target`:$AppDir/scripts/vps/"
if ($LASTEXITCODE -ne 0) { throw "scp failed for miya-online-watch.sh" }
scp $localService "$target`:/tmp/"
if ($LASTEXITCODE -ne 0) { throw "scp failed for miya-online-watch.service" }
scp $localTimer "$target`:/tmp/"
if ($LASTEXITCODE -ne 0) { throw "scp failed for miya-online-watch.timer" }
scp $localInstall "$target`:/tmp/"
if ($LASTEXITCODE -ne 0) { throw "scp failed for install-miya-online-watch-systemd-remote.sh" }

Write-Host "Installing systemd units on remote host (sudo; TTY for password) ..."
$remoteCmd = @"
set -euo pipefail
cd "$AppDir"
chmod +x scripts/vps/miya-online-watch.sh
chmod +x /tmp/install-miya-online-watch-systemd-remote.sh
sudo /tmp/install-miya-online-watch-systemd-remote.sh
ls -la /var/log/muhlya-reports/online_candles_latest.* || true
"@

ssh -tt $target $remoteCmd
if ($LASTEXITCODE -ne 0) { throw "remote installation failed" }

Write-Host ""
Write-Host "Done. Miya online watch deployment completed."
