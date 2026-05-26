# Examples

本目錄包含 mm-web 插件的完整使用示例，包括 Server 和 Client。

## 項目結構

```
examples/
├── README.md           # 本文件
├── main.go            # Server + Client 完整測試
├── go.mod             # Go 模塊定義
└── client/
    └── main.go        # 獨立 Client 示例
```

## 快速開始

### 1. 安裝依賴

```bash
cd examples
go mod tidy
```

### 2. 運行完整測試

```bash
go run main.go
```

這將：
- 啟動服務器在 `http://localhost:8080`
- 自動執行 JSONC 和 MetaMessage 格式的 API 測試
- 顯示測試結果

### 3. 獨立運行客戶端

如果服務器已在運行，可以單獨運行客戶端：

```bash
go run client/main.go
```

## 測試內容

### JSONC 格式測試

```bash
# 獲取用戶列表
curl http://localhost:8080/api/v1/users

# 獲取單個用戶
curl http://localhost:8080/api/v1/users/1

# 創建用戶
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/jsonc" \
  -H "Accept: application/jsonc" \
  -d '{"name":"David","email":"david@example.com","age":28}'

# 更新用戶
curl -X PUT http://localhost:8080/api/v1/users/1 \
  -H "Content-Type: application/jsonc" \
  -H "Accept: application/jsonc" \
  -d '{"name":"Alice Updated"}'

# 刪除用戶
curl -X DELETE http://localhost:8080/api/v1/users/3
```

### MetaMessage 格式測試

```bash
# 獲取用戶列表 (MetaMessage)
curl http://localhost:8080/api/v1/users \
  -H "Accept: application/x-metamessage"

# 創建用戶 (MetaMessage)
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/x-metamessage" \
  -H "Accept: application/x-metamessage" \
  -d '{"name":"David","email":"david@example.com","age":28}'
```

## API 端點

| 方法 | 路徑 | 描述 |
|------|------|------|
| GET | /health | 健康檢查 |
| GET | /api/v1/users | 獲取用戶列表 |
| GET | /api/v1/users/:id | 獲取單個用戶 |
| POST | /api/v1/users | 創建用戶 |
| PUT | /api/v1/users/:id | 更新用戶 |
| DELETE | /api/v1/users/:id | 刪除用戶 |

## 使用 Client SDK

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/metamessage/gin-mm"
)

func main() {
    ctx := context.Background()
    client := mm.NewClient("http://localhost:8080")

    // 設置格式 (JSONC 或 MetaMessage)
    client.SetJSONC()
    // client.SetMetaMessage()

    // 獲取用戶列表
    users, err := client.ListUsers(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("用戶總數: %d\n", users.Total)

    // 創建用戶
    resp, err := client.CreateUser(ctx, &mm.CreateUserRequest{
        Name:  "Test User",
        Email: "test@example.com",
        Age:   25,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("創建結果: %s\n", resp.Message)
}
```

## 輸出示例

```
==============================================================
🧪 開始測試...
==============================================================

📌 測試 1: 使用 JSONC 格式

  📋 GET /api/v1/users
  ✅ 成功! Total: 3
     - ID: 1, Name: Alice, Email: alice@example.com, Age: 25
     - ID: 2, Name: Bob, Email: bob@example.com, Age: 30
     - ID: 3, Name: Charlie, Email: charlie@example.com, Age: 35

  📋 GET /api/v1/users/1
  ✅ 成功! Message: success
     User: map[age:25 email:alice@example.com id:1 is_active:true name:Alice]

  📋 POST /api/v1/users
  ✅ 成功! Message: user created
     New User: map[age:28 email:david@example.com id:4 is_active:true name:David]

  ...

==============================================================
✨ 所有測試完成!
==============================================================
```

## 故障排除

### 端口被佔用

如果 `8080` 端口被佔用，可以修改 `main.go` 中的端口：

```go
func main() {
    const port = "8081"  // 改為其他端口
    runServer(port)
    // ...
}
```

### 連接失敗

確保服務器已啟動並監聽正確的端口：

```bash
# 檢查端口
lsof -i :8080

# 或測試連接
curl http://localhost:8080/health
```
