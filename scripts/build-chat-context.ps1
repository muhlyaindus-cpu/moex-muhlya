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

# Записываем header с UTF-8 (без BOM)
$utf8NoBom = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllText($outPath, $header, $utf8NoBom)

$stylePath = Join-Path $root $styleFile
if (Test-Path $stylePath) {
    $content = [System.IO.File]::ReadAllText($stylePath, [System.Text.Encoding]::UTF8)
    [System.IO.File]::AppendAllText($outPath, "`n---`n`n## FILE: $styleFile`n`n$content", $utf8NoBom)
}

foreach ($f in $files) {
    $path = Join-Path $root $f
    if (Test-Path $path) {
        $content = [System.IO.File]::ReadAllText($path, [System.Text.Encoding]::UTF8)
        [System.IO.File]::AppendAllText($outPath, "`n---`n`n## FILE: $f`n`n$content", $utf8NoBom)
    }
}

Write-Host "Built context packet:" $outPath

