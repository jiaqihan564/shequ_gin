### 历史头像接口对接说明

- **接口地址**
  - GET `/api/user/avatar/history`
  - 兼容别名：GET `/api/avatar/history`

- **鉴权**
  - 需要 `Authorization: Bearer <token>`（JWT）。返回的是当前登录用户的历史头像。

- **请求方式与参数**
  - 方法：GET
  - 查询参数：无（服务端固定按时间倒序，最多返回 50 条）

- **成功响应（200）**
```json
{
  "code": 200,
  "message": "OK",
  "requestID": "xxxxxx",
  "data": {
    "items": [
      {
        "key": "username/1726135248.png",
        "url": "https://assets.example.com/username/1726135248.png",
        "size": 12345,
        "last_modified": 1726135248
      }
      // ... 最多 50 条，按时间倒序（新→旧）
    ]
  }
}
```

- **失败响应**
  - 结构遵循全局规范（`code`/`message`/`requestID`，必要时附 `errorCode`）。

- **列表说明**
  - 仅包含历史归档文件：`username/{timestamp}.png`
  - 当前头像 `username/avatar.png` 不在列表中

- **历史保留策略（前端提示用）**
  - 每次上传后，旧头像会被归档为 `username/{timestamp}.png`
  - 系统仅保留最近 9 个历史头像，多余的自动删除（异步执行）
  - 当前头像固定为 `username/avatar.png`

- **示例请求**
```bash
curl -H "Authorization: Bearer <token>" \
  https://your-domain.com/api/user/avatar/history
```


