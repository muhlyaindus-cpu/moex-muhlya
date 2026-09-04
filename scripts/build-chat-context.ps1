Param(
    [string]$OutputFile = "..\CHAT_CONTEXT_PACKET.md"
)

$root = Split-Path -Parent $PSScriptRoot
$outPath = Join-Path $PSScriptRoot $OutputFile

# Новосибирск: UTC+7, без перехода на летнее время (для единообразия с торговой логикой терминала).
$generatedNsk = [DateTime]::UtcNow.AddHours(7).ToString("yyyy-MM-dd HH:mm:ss")

$styleFile = "COLLAB_STYLE.md"
$files = @(
    "CHAT_START_PROMPT.md",
    "NEXT_STEPS.md",
    "SESSION_LOG.md",
    "docs/MIYA_MUHLYA_ROLES.md",
    "CHAT_HANDOFF_TEMPLATE.md"
)

$header = @"
# CHAT CONTEXT PACKET

Generated: $generatedNsk (NSK, UTC+7)

Этот файл можно целиком скопировать в новый чат.
Он перезаписывается этим скриптом:
`.\scripts\build-chat-context.ps1`

---

## QUICK START (для нового чата)

1. Скопируй весь этот файл в новый чат.
2. Попроси агента сначала прочитать разделы:
   - `COLLAB_STYLE`
   - `NEXT_STEPS`
   - `SESSION_LOG`
3. Работай по правилу: код только после команды `opengo`.
4. Роли Мухля / Мия — в секции `FILE: docs/MIYA_MUHLYA_ROLES.md` ниже.

---

## Цель проекта и нашего общения

Создать торговую площадку (терминал и торговый ассистент), которая будет вести операции с ценными бумагами на Московской бирже и **стремиться увеличить торговый счёт в рублях**. Прибыль не гарантируется; приоритет — контроль риска, прозрачность логики и проверка на истории/бумаге перед боем.

"@

Set-Content -Path $outPath -Value $header -Encoding UTF8

$stylePath = Join-Path $root $styleFile
if (Test-Path $stylePath) {
    Add-Content -Path $outPath -Value "`n---`n`n## FILE: $styleFile`n"
    Get-Content $stylePath | Add-Content -Path $outPath
}

foreach ($f in $files) {
    $path = Join-Path $root $f
    if (Test-Path $path) {
        Add-Content -Path $outPath -Value "`n---`n`n## FILE: $f`n"
        Get-Content $path | Add-Content -Path $outPath
    }
}

Write-Host "Built context packet:" $outPath

