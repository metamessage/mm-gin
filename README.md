# mm-web-go

為 golang web 框架提供 MetaMessage 協議支持，提供編解碼、數據綁定、Schema 發現等功能。支持 gin、echo、fiber、chi、net/http。

**[中文](README.md)** | [English](README.en.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

## 特性

- **一行初始化**：`mmweb.Init(r, "/api/v1")` 同時註冊中間件與路由組
- **泛型路由註冊**：`mmweb.POST[T](path, handler)` 自動類型安全的請求綁定，`GET`/`DELETE` 同樣支持
- **Schema 發現**：所有泛型路由自動註冊 OPTIONS 端點，便於客戶端請求驗證
- **請求解碼**：自動檢測並解碼 JSONC 與 MetaMessage 二進制格式的請求體
- **響應編碼**：自動將響應數據編碼為 MetaMessage 二進制格式
- **查詢參數綁定**：GET/DELETE 通過 `?data=<hex>` 查詢參數綁定 MetaMessage 數據
- **HTTP 客戶端**：泛型 `DoRequest[REQ, RESP]` 搭配 Schema 預檢驗證

## 安裝

```bash
go get github.com/metamessage/mm-web-go
```

## 快速開始

### 服務端

```go
package main

import (
    "github.com/gin-gonic/gin"
    server "github.com/metamessage/mm-web-go/mmgin"
)

type CreateUserRequest struct {
    Name  string `mm:"desc=User name; min=1; max=50"`
    Email string `mm:"type=email; desc=Email address"`
    Age   uint8  `mm:"desc=Age; min=0; max=150"`
}

type UserResponse struct {
    ID    int64  `mm:"desc=User ID"`
    Name  string `mm:"desc=User name"`
    Email string `mm:"type=email; desc=Email address"`
    Age   uint8  `mm:"desc=Age"`
}

func main() {
    r := gin.Default()

    // 一行初始化：中間件 + 路由組
    server.Init(r, "/api/v1")

    // 泛型路由，自動綁定並註冊 OPTIONS Schema 發現
    server.POST("/users", func(c *gin.Context, req *CreateUserRequest) (any, string, error) {
        return UserResponse{
            ID:    1,
            Name:  req.Name,
            Email: req.Email,
            Age:   req.Age,
        }, "", nil
    })

    r.Run(":8080")
}
```

### 客戶端

```go
package main

import (
	"fmt"
	"github.com/metamessage/mm-web-go/client"
)

func main() {
	client.SetDefaultClient("http://localhost:8080", false)

	// 泛型請求，包含 Schema 驗證
	req := CreateUserRequest{Name: "Alice", Email: "alice@example.com", Age: 25}
	resp, err := client.POST[CreateUserRequest, UserResponse]("/api/v1/users", &req)
	if err != nil {
		panic(err)
	}
	fmt.Printf("User: %+v\n", resp)
}
```

---

## 多框架適配

所有框架提供**完全一致的泛型路由註冊 API**，handler 簽名統一為 `func(ctx, *T) (any, string, error)`。

#### Gin

```go
import "github.com/gin-gonic/gin"
import server "github.com/metamessage/mm-web-go/mmgin"

r := gin.Default()
server.Init(r, "/api/v1")

// 泛型路由：自動綁定 + OPTIONS Schema 發現
server.GET("/users", func(c *gin.Context, req *any) (any, string, error) {
    return ListUsersResponse{...}, "", nil
})
server.GET("/users/:id", func(c *gin.Context, req *any) (any, string, error) {
    return APIResponse{...}, "", nil
})
server.POST("/users", func(c *gin.Context, req *CreateUserRequest) (any, string, error) {
    return UserResponse{...}, "", nil
})
server.PUT("/users/:id", func(c *gin.Context, req *UpdateUserRequest) (any, string, error) {
    return APIResponse{...}, "", nil
})
server.DELETE("/users/:id", func(c *gin.Context, req *any) (any, string, error) {
    return APIResponse{...}, "", nil
})
```

#### Echo

```go
import "github.com/labstack/echo/v4"
import server "github.com/metamessage/mm-web-go/mmecho"

e := echo.New()
server.Init(e, "/api/v1")

server.GET("/users", func(c echo.Context, req *any) (any, string, error) {
    return ListUsersResponse{...}, "", nil
})
server.POST("/users", func(c echo.Context, req *CreateUserRequest) (any, string, error) {
    return UserResponse{...}, "", nil
})
```

#### Fiber

```go
import "github.com/gofiber/fiber/v2"
import server "github.com/metamessage/mm-web-go/mmfiber"

app := fiber.New()
server.Init(app, "/api/v1")

server.GET("/users", func(c *fiber.Ctx, req *any) (any, string, error) {
    return ListUsersResponse{...}, "", nil
})
server.POST("/users", func(c *fiber.Ctx, req *CreateUserRequest) (any, string, error) {
    return UserResponse{...}, "", nil
})
```

#### Chi

```go
import "github.com/go-chi/chi/v5"
import server "github.com/metamessage/mm-web-go/mmchi"

r := chi.NewRouter()
server.Init(r, "/api/v1")

server.GET("/users", func(r *http.Request, req *any) (any, string, error) {
    return ListUsersResponse{...}, "", nil
})
server.POST("/users", func(r *http.Request, req *CreateUserRequest) (any, string, error) {
    return UserResponse{...}, "", nil
})
```

#### net/http (vanilla)

```go
import server "github.com/metamessage/mm-web-go/mmvanilla"

mux := http.NewServeMux()
server.Init(mux, "/api/v1")

server.GET("/users", func(r *http.Request, req *any) (any, string, error) {
    return ListUsersResponse{...}, "", nil
})
server.POST("/users", func(r *http.Request, req *CreateUserRequest) (any, string, error) {
    return UserResponse{...}, "", nil
})
```

> 非泛型路由（如 `HEAD`/`OPTIONS`/`Any`）也可通過 `server.HEAD()`、`server.OPTIONS()`、`server.Any()` 包級函數註冊，使用各框架原生 Handler 類型。

---

## API 文檔

### 一行初始化

#### Init

`Init` 會註冊解碼與編碼中間件、創建路由組，並將其設為後續所有路由註冊的默認組。

```go
rg := mmgin.Init(r, "/api/v1")
// 此後 mmgin.GET/POST/PUT/DELETE 等都使用此組
```

---

### 路由註冊

#### 泛型 GET / DELETE

具備自動查詢參數綁定與 OPTIONS Schema 發現的泛型路由註冊。通過 `?data=<hex>` 查詢參數傳遞 MetaMessage 編碼的請求數據。

```go
// Handler[T] 定義：func(c *gin.Context, req *T) (any, string, error)
type Handler[T any] func(c *gin.Context, req *T) (data any, tag string, err error)

mmgin.GET("/users", func(c *gin.Context, req *any) (any, string, error) {
    return ListUsersResponse{...}, "", nil
})

mmgin.DELETE("/users/:id", func(c *gin.Context, req *any) (any, string, error) {
    return APIResponse{...}, "", nil
})
```

#### 泛型 POST / PUT / PATCH

具備自動請求體綁定與 OPTIONS Schema 發現的泛型路由註冊：

```go
mmgin.POST("/users", func(c *gin.Context, req *CreateUserRequest) (any, string, error) {
    return UserResponse{...}, "", nil
})

mmgin.PUT("/users/:id", func(c *gin.Context, req *UpdateUserRequest) (any, string, error) {
    return APIResponse{...}, "", nil
})
```

每個泛型路由會自動在同一路徑註冊 OPTIONS 端點。客戶端可發送 OPTIONS 請求獲取請求結構體的 Schema（以 MetaMessage 二進制編碼）。

#### HEAD / OPTIONS / Any

不需要自動綁定的方法的標準路由註冊，使用各框架原生 Handler 類型：

```go
mmgin.HEAD("/health", healthCheck)
mmgin.OPTIONS("/resources", optionsHandler)
mmgin.Any("/catch-all", catchAllHandler)
```

---

### 響應函數

#### Respond

為編碼中間件設置響應數據。接受可選的 mm 標籤字符串：

```go
mmgin.Respond(c, User{Name: "Alice"}, "")
mmgin.Respond(c, users, "desc=User list response")
```

#### RespondWithStatus

設置響應數據與自定義 HTTP 狀態碼：

```go
mmgin.RespondWithStatus(c, http.StatusCreated, APIResponse{
    Code:    0,
    Message: "user created",
    Data:    &newUser,
}, "")
```

#### AbortWithMetaMessage

發送 MetaMessage 格式的錯誤響應並終止請求：

```go
mmgin.AbortWithMetaMessage(c, http.StatusNotFound, ErrorResponse{
    Error: "user not found",
})
```

---

### 數據綁定

#### Bind

將請求體綁定到結構體（自動檢測格式）：

```go
var user User
if err := mmgin.Bind(c, &user); err != nil {
    // 處理錯誤
}
```

#### BindQuery

將查詢參數綁定到結構體（從 `?data=<hex>` 讀取 MetaMessage 編碼數據）：

```go
var filter Filter
if err := mmgin.BindQuery(c, &filter); err != nil {
    // 處理錯誤
}
```

#### ShouldBind / ShouldBindWithTag

不發送錯誤響應的變體，直接返回錯誤：

```go
var user User
if err := mmgin.ShouldBind(c, &user); err != nil {
    // 手動處理錯誤
}
```

#### MustBindAndValidate

綁定並驗證，失敗時自動返回錯誤響應：

```go
var req CreateUserRequest
if err := mmgin.MustBindAndValidate(c, &req); err != nil {
    return // 錯誤響應已自動發送
}
```

#### Validator 接口

實現 `Validator` 接口以添加自定義驗證邏輯：

```go
type CreateUserRequest struct {
    Name string `mm:"desc=User name"`
    Age  uint8  `mm:"desc=Age"`
}

func (r *CreateUserRequest) Validate() error {
    if r.Age < 18 {
        return fmt.Errorf("user must be at least 18 years old")
    }
    return nil
}
```

---

### HTTP 客戶端

`client` 包提供用於 MetaMessage 通信的泛型 HTTP 客戶端。

#### Client

```go
c := client.NewClient("http://localhost:8080", false)
```

#### SetDefaultClient

設置全局默認客戶端：

```go
client.SetDefaultClient("http://localhost:8080", false)
```

#### DoRequest

以類型安全的請求/響應執行泛型請求。會自動先發送 OPTIONS 預檢請求驗證 Schema：

```go
resp, err := client.DoRequest[CreateUserRequest, UserResponse](
    c, "POST", "/api/v1/users", req,
)
```

#### 便捷函數

使用默認客戶端的包級別便捷函數：

```go
client.GET[any, ListUsersResponse]("/api/v1/users", nil)
client.POST[CreateUserRequest, UserResponse]("/api/v1/users", req)
client.PUT[UpdateUserRequest, APIResponse]("/api/v1/users/1", req)
client.DELETE[any, APIResponse]("/api/v1/users/1", nil)
client.PATCH[UpdateUserRequest, APIResponse]("/api/v1/users/1", req)
```

---

### Schema 發現

服務端的泛型路由（GET/POST/PUT/DELETE/PATCH）會自動註冊 OPTIONS 端點用於 Schema 發現。OPTIONS 響應會返回一個 MetaMessage 編碼的結構體實例，其中包含完整的類型、約束條件與描述元數據。

客戶端在發送實際請求前，會自動利用此機制進行請求驗證：

```
客戶端                          服務端
  │                               │
  ├── OPTIONS /api/v1/users ──────►
  │◄──── MetaMessage Schema ──────┤
  │     (struct definition)       │
  │                               │
  ├── POST /api/v1/users ────────►
  │◄──── MetaMessage Response ────┤
```

---

## 示例

完整的服務端 + 客戶端示例請見 [examples](examples/)。

```bash
cd examples/gin    # 或 echo / fiber / chi / vanilla
go run main.go
```

此示例展示：

- 使用 `Init()` 及泛型路由註冊的服務端
- GET/DELETE 通過 `?data=<hex>` 查詢參數傳遞請求數據
- 使用 MetaMessage 二進制協議的 CRUD 操作
- 通過 OPTIONS 預檢請求進行 Schema 驗證的客戶端

---

## 依賴

- [metamessage/metamessage](https://github.com/metamessage/metamessage) - MetaMessage 協議實現

## 許可證

MIT

---

**[中文](README.md)** | [English](README.en.md) | [日本語](README.ja.md) | [한국어](README.ko.md)
