# 测试更新用户信息
Write-Host "重新登录获取新token..."
$loginBody = @{
    username = "testuser"
    password = "password123"
} | ConvertTo-Json

try {
    $loginResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/login" -Method POST -ContentType "application/json" -Body $loginBody
    Write-Host "登录成功，获取到新token"
    
    $token = $loginResponse.data.token
    $headers = @{
        "Authorization" = "Bearer $token"
        "Content-Type" = "application/json"
    }
    
    Write-Host "`n测试更新用户信息..."
    $updateBody = @{
        email = "updated@example.com"
    } | ConvertTo-Json
    
    try {
        $updateResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/user/profile" -Method PUT -Headers $headers -Body $updateBody
        Write-Host "更新用户信息成功:"
        $updateResponse | ConvertTo-Json -Depth 3
    } catch {
        Write-Host "更新用户信息失败:"
        Write-Host $_.Exception.Message
    }
    
} catch {
    Write-Host "登录失败:"
    Write-Host $_.Exception.Message
}
