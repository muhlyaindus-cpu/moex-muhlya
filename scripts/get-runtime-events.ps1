param(
    [int]$Limit = 50,
    [string]$Container = "moex_trading_db",
    [string]$Database = "",
    [string]$User = "",
    [string]$StepLike = ""
)

if ($Limit -lt 1) { $Limit = 1 }
if ($Limit -gt 500) { $Limit = 500 }
if ([string]::IsNullOrWhiteSpace($Database)) { $Database = $(if ($env:PGDATABASE) { $env:PGDATABASE } else { "moex_trading" }) }
if ([string]::IsNullOrWhiteSpace($User)) { $User = $(if ($env:PGUSER) { $env:PGUSER } else { "postgres" }) }

$where = ""
if (-not [string]::IsNullOrWhiteSpace($StepLike)) {
    $escaped = $StepLike.Replace("'", "''")
    $where = "WHERE step ILIKE '%$escaped%'"
}

$sql = @"
SELECT to_char(at AT TIME ZONE 'Asia/Novosibirsk', 'YYYY-MM-DD HH24:MI:SS') AS at_nsk,
       level,
       step,
       details
FROM runtime_events
$where
ORDER BY created_at DESC
LIMIT $Limit;
"@

docker exec -i $Container psql -U $User -d $Database -c $sql
