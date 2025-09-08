# 测试需要认证的API
$token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjozLCJ1c2VybmFtZSI6InRlc3R1c2VyIiwiaXNzIjoidGVzdHVzZXIiLCJzdWIiOiIzIiwiYXVkIjpbImNvbW11bml0eS1hcGkiXSwiZXhwIjoxNzU3NDEzNzg2LCJuYmYiOjE3NTczMjczODYsImlhdCI6MTc1NzMyNzM4NiwianRpIjoiMyJ9.FsoiDdmb3VmsJYjFzF3lS2M-pHUO-9-YQL0RcQogDow"

$headers = @{
    "Authorization" = "Bearer $token"
    "Content-Type" = "application/json"
}

Write-Host "测试获取用户信息..."
try {
    $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/user/profile" -Method GET -Headers $headers
    Write-Host "获取用户信息成功:"
    $response | ConvertTo-Json -Depth 3
} catch {
    Write-Host "获取用户信息失败:"
    Write-Host $_.Exception.Message
}

Write-Host "`n测试更新用户信息..."
$updateBody = @{
    email = "newemail@example.com"
} | ConvertTo-Json

try {
    $updateResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/user/profile" -Method PUT -Headers $headers -Body $updateBody
    Write-Host "更新用户信息成功:"
    $updateResponse | ConvertTo-Json -Depth 3
} catch {
    Write-Host "更新用户信息失败:"
    Write-Host $_.Exception.Message
}

Write-Host "`n测试获取性能指标..."
try {
    $metricsResponse = Invoke-RestMethod -Uri "http://localhost:8080/metrics" -Method GET
    Write-Host "获取性能指标成功:"
    $metricsResponse | ConvertTo-Json -Depth 3
} catch {
    Write-Host "获取性能指标失败:"
    Write-Host $_.Exception.Message
}
