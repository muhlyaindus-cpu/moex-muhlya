# stop.ps1
Write-Host "🛑 Остановка MOEX Trading Terminal" -ForegroundColor Cyan

Write-Host "`n🐳 Остановка Docker контейнеров..." -ForegroundColor Yellow
docker-compose down

Write-Host "✅ Все сервисы остановлены" -ForegroundColor Green