# start.ps1
Write-Host "🚀 Запуск MOEX Trading Terminal" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════" -ForegroundColor Cyan

# Проверка Docker
Write-Host "`n📦 Проверка Docker..." -ForegroundColor Yellow
docker --version
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Docker не найден! Установите Docker Desktop" -ForegroundColor Red
    exit 1
}

# Запуск Docker Compose
Write-Host "`n🐳 Запуск PostgreSQL в Docker..." -ForegroundColor Yellow
docker-compose up -d

# Ожидание готовности PostgreSQL
Write-Host "⏳ Ожидание запуска PostgreSQL..." -ForegroundColor Yellow
Start-Sleep -Seconds 5

# Проверка PostgreSQL
Write-Host "`n🔍 Проверка подключения к PostgreSQL..." -ForegroundColor Yellow
docker exec moex_trading_db pg_isready -U trader
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ PostgreSQL готов к работе" -ForegroundColor Green
} else {
    Write-Host "❌ Ошибка подключения к PostgreSQL" -ForegroundColor Red
}

# Запуск Go приложения
Write-Host "`n🚀 Запуск Go приложения..." -ForegroundColor Yellow
go run cmd/main.go