# 测试用户注册API
$body = @{
    username = "testuser"
    password = "password123"
    email = "test@example.com"
} | ConvertTo-Json

Write-Host "测试用户注册..."
try {
    $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/register" -Method POST -ContentType "application/json" -Body $body
    Write-Host "注册成功:"
    $response | ConvertTo-Json -Depth 3
} catch {
    Write-Host "注册失败:"
    Write-Host $_.Exception.Message
}

Write-Host "`n测试用户登录..."
$loginBody = @{
    username = "testuser"
    password = "password123"
} | ConvertTo-Json

try {
    $loginResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/login" -Method POST -ContentType "application/json" -Body $loginBody
    Write-Host "登录成功:"
    $loginResponse | ConvertTo-Json -Depth 3
} catch {
    Write-Host "登录失败:"
    Write-Host $_.Exception.Message
}
