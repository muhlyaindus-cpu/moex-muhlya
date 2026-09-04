#Requires -Version 5.1
# Создаёт ярлык на рабочем столе: MOEX Trading Terminal → scripts\launch-moex-trader-gui.ps1
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$Root = Split-Path -Parent $ScriptDir
$Launch = Join-Path $ScriptDir "launch-moex-trader-gui.ps1"
$Gui = Join-Path $Root "moex-trader-gui"
$Exe = Join-Path $Gui "build\bin\moex-trader-gui.exe"

if (-not (Test-Path -LiteralPath $Launch)) {
    Write-Error "Не найден файл: $Launch"
    exit 1
}

$Desktop = [Environment]::GetFolderPath("Desktop")
$LinkPath = Join-Path $Desktop "MOEX Trading Terminal.lnk"

$Wsh = New-Object -ComObject WScript.Shell
$Shortcut = $Wsh.CreateShortcut($LinkPath)
$Shortcut.TargetPath = "powershell.exe"
$Shortcut.Arguments = "-ExecutionPolicy Bypass -NoProfile -File `"$Launch`""
$Shortcut.WorkingDirectory = $Gui
$Shortcut.Description = "MOEX Trading Terminal (Finam + MOEX)"
if (Test-Path -LiteralPath $Exe) {
    $Shortcut.IconLocation = "$Exe,0"
}
$Shortcut.Save()

Write-Host "Ярлык создан: $LinkPath" -ForegroundColor Green
