Param(
    [string]$TranscriptsPath = "C:\Users\Администратор\.cursor\projects\c-Users-MyProgram-Trading\agent-transcripts",
    [string]$BuildScriptPath = ".\scripts\build-chat-context.ps1"
)

$root = Split-Path -Parent $PSScriptRoot
$buildScript = Join-Path $root (Split-Path $BuildScriptPath -Leaf)
if (-not (Test-Path $buildScript)) {
    $buildScript = Join-Path $root "scripts\build-chat-context.ps1"
}

if (-not (Test-Path $TranscriptsPath)) {
    Write-Host "Transcripts path not found: $TranscriptsPath"
    exit 1
}

Write-Host "Watching transcripts folder:"
Write-Host "  $TranscriptsPath"
Write-Host "On each change, rebuilding CHAT_CONTEXT_PACKET.md..."

function Get-NskTimestamp {
    [DateTime]::UtcNow.AddHours(7).ToString("yyyy-MM-dd HH:mm:ss")
}

$watcher = New-Object System.IO.FileSystemWatcher
$watcher.Path = $TranscriptsPath
$watcher.Filter = "*.jsonl"
$watcher.IncludeSubdirectories = $false
$watcher.EnableRaisingEvents = $true

$action = {
    try {
        & "$using:buildScript" | Out-Null
        Write-Host "[$(Get-NskTimestamp) NSK] Context packet updated."
    } catch {
        Write-Host "[$(Get-NskTimestamp) NSK] Update failed: $($_.Exception.Message)"
    }
}

$created = Register-ObjectEvent $watcher Created -Action $action
$changed = Register-ObjectEvent $watcher Changed -Action $action
$renamed = Register-ObjectEvent $watcher Renamed -Action $action

try {
    while ($true) {
        Start-Sleep -Seconds 2
    }
}
finally {
    Unregister-Event -SourceIdentifier $created.Name -ErrorAction SilentlyContinue
    Unregister-Event -SourceIdentifier $changed.Name -ErrorAction SilentlyContinue
    Unregister-Event -SourceIdentifier $renamed.Name -ErrorAction SilentlyContinue
    $watcher.Dispose()
}
