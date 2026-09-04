#Requires -Version 5.1
# Запуск MOEX Trading Terminal (moex-trader-gui): релизный .exe, если есть, иначе wails dev.
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$Root = Split-Path -Parent $ScriptDir
$Gui = Join-Path $Root "moex-trader-gui"
$Exe = Join-Path $Gui "build\bin\moex-trader-gui.exe"

if (-not (Test-Path -LiteralPath $Gui)) {
    Write-Error "Не найден каталог проекта: $Gui"
    exit 1
}

if (Test-Path -LiteralPath $Exe) {
    Start-Process -FilePath $Exe -WorkingDirectory $Gui
    exit 0
}

if (-not (Get-Command wails -ErrorAction SilentlyContinue)) {
    Write-Host "Wails CLI не найден в PATH. Установите: https://wails.io/docs/gettingstarted/installation" -ForegroundColor Yellow
    Write-Host "Или соберите приложение из moex-trader-gui: wails build" -ForegroundColor Yellow
    Read-Host "Enter для выхода"
    exit 1
}

$cmd = "Set-Location -LiteralPath '$($Gui.Replace("'", "''"))'; wails dev"
Start-Process -FilePath "powershell.exe" -ArgumentList @("-NoExit", "-ExecutionPolicy", "Bypass", "-NoProfile", "-Command", $cmd) -WorkingDirectory $Gui
