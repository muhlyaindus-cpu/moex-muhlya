<#
.SYNOPSIS
    Чеклист еженедельной ревизии статусных документов (см. COLLAB_STYLE.md, NEXT_STEPS.md).

.PARAMETER RebuildContext
    После чеклиста запустить build-chat-context.ps1 (пересборка CHAT_CONTEXT_PACKET.md).
#>
Param(
    [switch]$RebuildContext
)

$root = Split-Path -Parent $PSScriptRoot
# Как в build-chat-context.ps1: NSK UTC+7 для единообразия с торговой логикой.
$whenNsk = [DateTime]::UtcNow.AddHours(7).ToString("yyyy-MM-dd HH:mm:ss")

Write-Host ""
Write-Host "=== Еженедельная ревизия документов ===" -ForegroundColor Cyan
Write-Host "Время (NSK): $whenNsk"
Write-Host "Корень проекта: $root"
Write-Host ""

$items = @(
    "SESSION_LOG.md — сверху есть факты за последнюю ~неделю (деплой, env, инциденты)?",
    "NEXT_STEPS.md — блок «Оперативный фокус» по-прежнему правда? Устаревшее → ## Архив с датой, не удалять.",
    "VPS / боевой .env (вне git) — сверить с архивом и фокусом; расхождение → правка доков или явная пометка в SESSION_LOG.",
    "CHAT_CONTEXT_PACKET.md — при необходимости: .\scripts\build-chat-context.ps1 (или этот скрипт с -RebuildContext).",
    "По желанию: просмотреть git-лог недели — нет ли важного только в коммитах без записи в SESSION_LOG."
)

$i = 1
foreach ($line in $items) {
    Write-Host ("  {0}. {1}" -f $i, $line)
    $i++
}

Write-Host ""
if ($RebuildContext) {
    $build = Join-Path $PSScriptRoot "build-chat-context.ps1"
    & $build
} else {
    Write-Host "Подсказка: пересобрать пакет для нового чата:" -ForegroundColor DarkGray
    Write-Host "  .\scripts\weekly-doc-review.ps1 -RebuildContext" -ForegroundColor DarkGray
    Write-Host "  или: .\scripts\build-chat-context.ps1" -ForegroundColor DarkGray
}
