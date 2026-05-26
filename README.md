# mm-web

Gin 框架的 MetaMessage 协议插件，提供编解码、数据绑定、Schema 发现以及泛型路由注册功能。

**[中文](README.md)** | [English](README.en.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

## 特性

- **一行初始化**：`mmgin.Init(r, "/api/v1")` 同时注册中间件与路由组
- **泛型路由注册**：`mmgin.POST[T](path, handler)` 自动类型安全的请求绑定
- **Schema 发现**：POST/PUT/PATCH 自动注册 OPTIONS，便于客户端请求验证
- **请求解码**：自动检测并解码 JSONC 与 MetaMessage 二进制格式的请求体
- **响应编码**：自动将响应数据编码为 MetaMessage 二进制格式
- **数据绑定**：支持请求体、查询参数、URI 参数及请求头绑定
- **自定义验证**：通过 `Validator` 接口支持结构级别的自定义验证逻辑
- **HTTP 客户端**：泛型 `DoRequest[REQ, RESP]` 搭配 Schema 预检验证

## 安装

```bash
go get github.com/metamessage/mm-web
```

## 快速开始

### 服务端

```go
package main

import (
    "github.com/gin-gonic/gin"
    mmgin "github.com/metamessage/mm-web"
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

    // 一行初始化：中间件 + 路由组
    mmgin.Init(r, "/api/v1")

    // 泛型路由，自动绑定并注册 OPTIONS Schema 发现
    mmgin.POST("/users", func(c *gin.Context, req *CreateUserRequest) {
        mmgin.Respond(c, UserResponse{
            ID:    1,
            Name:  req.Name,
            Email: req.Email,
            Age:   req.Age,
        }, "")
    })

    r.Run(":8080")
}
```

### 客户端

```go
package main

import (
	"fmt"
	"github.com/metamessage/mm-web/client"
)

func main() {
	client.SetDefaultClient("http://localhost:8080")

	// 泛型请求，包含 Schema 验证
	req := CreateUserRequest{Name: "Alice", Email: "alice@example.com", Age: 25}
	resp, err := client.POST[CreateUserRequest, UserResponse]("/api/v1/users", req)
	if err != nil {
		panic(err)
	}
	fmt.Printf("User: %+v\n", resp)
}
```

---

### 多框架适配

安装统一适配包：

```bash
go get github.com/metamessage/mm-web/server
```

`server` 包提供一套统一的 API，通过 `Init` 自动适配不同 Web 框架。**POST/PUT/PATCH 的处理逻辑完全可跨框架复用**，只需更换框架类型与原生路由方法。

```go
import "github.com/metamessage/mm-web/server"
```

#### Gin

```go
import "github.com/gin-gonic/gin"

r := gin.Default()
server.Init(r, "/api/v1")

// 使用 Gin 原生 API 注册 GET/DELETE 等
r.GET("/users", listUsers)
r.GET("/users/:id", getUser)
r.DELETE("/users/:id", deleteUser)

// 使用统一泛型 API 注册 POST/PUT/PATCH（自动绑定 + OPTIONS Schema 发现）
server.POST("/users", func(r *http.Request, req *CreateUserRequest) (any, error) {
	return UserResponse{ID: 1, Name: req.Name, Email: req.Email, Age: req.Age}, nil
})
server.PUT("/users/:id", func(r *http.Request, req *UpdateUserRequest) (any, error) {
	return APIResponse{Message: "updated"}, nil
})
```

#### Echo

```go
import "github.com/labstack/echo/v4"

e := echo.New()
server.Init(e, "/api/v1")

// Echo 原生 API
e.GET("/users", listUsers)

// 同一泛型处理器，框架无关
server.POST("/users", func(r *http.Request, req *CreateUserRequest) (any, error) {
	return UserResponse{ID: 1, Name: req.Name, Email: req.Email, Age: req.Age}, nil
})
```

#### Fiber

```go
import "github.com/gofiber/fiber/v2"

app := fiber.New()
server.Init(app, "/api/v1")

// Fiber 原生 API
app.Get("/users", listUsers)

// 同一泛型处理器
server.POST("/users", func(r *http.Request, req *CreateUserRequest) (any, error) {
	return UserResponse{ID: 1, Name: req.Name, Email: req.Email, Age: req.Age}, nil
})
```

#### Chi

```go
import "github.com/go-chi/chi/v5"

r := chi.NewRouter()
server.Init(r, "/api/v1")

// Chi 原生 API
r.Get("/users", listUsers)

// 同一泛型处理器
server.POST("/users", func(r *http.Request, req *CreateUserRequest) (any, error) {
	return UserResponse{ID: 1, Name: req.Name, Email: req.Email, Age: req.Age}, nil
})
```

#### net/http

```go
mux := http.NewServeMux()
server.Init(mux, "/api/v1")

// 标准库原生 API
mux.HandleFunc("/api/v1/users", listUsers)

// 同一泛型处理器
server.POST("/users", func(r *http.Request, req *CreateUserRequest) (any, error) {
	return UserResponse{ID: 1, Name: req.Name, Email: req.Email, Age: req.Age}, nil
})
```

> 框架特定的 `GET`、`HEAD`、`DELETE`、`Any` 等路由也可通过 `server.GET()`、`server.DELETE()` 等包级函数注册，接收各框架的原生 Handler 类型。

```go
// 使用包级函数统一注册非泛型路由（handler 类型需适配当前框架）
server.GET("/users", listUsers)
server.DELETE("/users/:id", deleteUser)
server.HEAD("/health", healthCheck)
server.OPTIONS("/resources", optionsHandler)
server.Any("/catch-all", catchAllHandler)
```

## API 文档

### 一行初始化

#### Init

`Init` 会注册 `MetaMessageDecoder` 与 `MetaMessageEncoder` 中间件、创建路由组，并将其设为后续所有路由注册的默认组。

```go
rg := mmgin.Init(r, "/api/v1")
// 此后 mmgin.GET/POST/PUT/DELETE 等都使用此组
```

---

### 路由注册

#### GET / HEAD / DELETE / OPTIONS / Any

不需要自动绑定的方法的标准路由注册：

```go
mmgin.GET("/users", listUsers)
mmgin.GET("/users/:id", getUser)
mmgin.DELETE("/users/:id", deleteUser)
mmgin.HEAD("/health", healthCheck)
mmgin.OPTIONS("/resources", optionsHandler)
mmgin.Any("/catch-all", catchAllHandler)
```

#### 泛型 POST / PUT / PATCH

具备自动请求绑定与 OPTIONS Schema 发现的泛型路由注册：

```go
// Handler[T any] 定义：func(c *gin.Context, req *T)
type Handler[T any] func(c *gin.Context, req *T)

mmgin.POST("/users", func(c *gin.Context, req *CreateUserRequest) {
    // req 已自动绑定并验证
    mmgin.Respond(c, UserResponse{...}, "")
})

mmgin.PUT("/users/:id", func(c *gin.Context, req *UpdateUserRequest) {
    mmgin.Respond(c, APIResponse{...}, "")
})
```

每个 POST/PUT/PATCH 路由会自动在同一路径注册 OPTIONS 端点。客户端可发送 OPTIONS 请求获取请求结构体的 Schema（以 MetaMessage 二进制编码）。

---

### 中间件

#### MetaMessageDecoder

支持 JSONC 与 MetaMessage 二进制格式的请求体解码中间件。

```go
// 使用默认配置
r.Use(mmgin.MetaMessageDecoder(nil))

// 自定义配置
config := &mmgin.DecodeConfig{
    AllowJSONC:       true,
    AllowMetaMessage: true,
    DefaultFormat:    mmgin.FormatAuto,
    MaxBodySize:      10 << 20, // 10MB
}
r.Use(mmgin.MetaMessageDecoder(config))
```

#### MetaMessageEncoder

将处理器设置的响应数据编码为 MetaMessage 二进制格式的响应编码中间件。

```go
// 使用默认配置
r.Use(mmgin.MetaMessageEncoder(nil))

// 自定义配置
config := &mmgin.EncodeConfig{
    DefaultFormat: mmgin.FormatMetaMessage,
    AutoNegotiate: false,
    SuccessCode:   http.StatusOK,
}
r.Use(mmgin.MetaMessageEncoder(config))
```

---

### 数据绑定

#### Bind

将请求体绑定到结构体（自动检测格式）：

```go
var user User
if err := mmgin.Bind(c, &user); err != nil {
    // 处理错误
}
```

#### BindWithTag

使用指定的 mm 标签绑定请求体：

```go
var user User
if err := mmgin.BindWithTag(c, &user, "custom_tag"); err != nil {
    // 处理错误
}
```

#### MustBind

绑定数据，失败时自动返回 400 错误响应：

```go
var user User
if err := mmgin.MustBind(c, &user); err != nil {
    return // 错误响应已自动发送
}
```

#### ShouldBind / ShouldBindWithTag

不发送错误响应的变体，直接返回错误：

```go
var user User
if err := mmgin.ShouldBind(c, &user); err != nil {
    // 手动处理错误
}
```

#### BindQuery

将查询参数绑定到结构体：

```go
var filter Filter
if err := mmgin.BindQuery(c, &filter); err != nil {
    // 处理错误
}
```

#### BindHeader

将请求头绑定到结构体：

```go
var headers Headers
if err := mmgin.BindHeader(c, &headers); err != nil {
    // 处理错误
}
```

#### BindUri

将 URI 参数绑定到结构体：

```go
var params Params
if err := mmgin.BindUri(c, &params); err != nil {
    // 处理错误
}
```

#### AutoBind

从所有来源自动绑定（优先级：URI 参数 > 查询参数 > 请求体）：

```go
var req Request
if err := mmgin.AutoBind(c, &req); err != nil {
    // 处理错误
}
```

---

### 数据验证

#### Validator 接口

实现 `Validator` 接口以添加自定义验证逻辑：

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

#### BindAndValidate

绑定并验证数据：

```go
var req CreateUserRequest
if err := mmgin.BindAndValidate(c, &req); err != nil {
    // 处理错误
}
```

#### MustBindAndValidate

绑定并验证，失败时自动返回错误响应：

```go
var req CreateUserRequest
if err := mmgin.MustBindAndValidate(c, &req); err != nil {
    return // 错误响应已自动发送
}
```

---

### 响应函数

#### Respond

为编码中间件设置响应数据。接受可选的 mm 标签字符串：

```go
mmgin.Respond(c, User{Name: "Alice"}, "")
mmgin.Respond(c, users, "desc=User list response")
```

#### RespondWithStatus

设置响应数据与自定义 HTTP 状态码：

```go
mmgin.RespondWithStatus(c, http.StatusCreated, APIResponse{
    Code:    0,
    Message: "user created",
    Data:    &newUser,
}, "")
```

#### SetMMResponse

直接设置响应数据（与 gin 的 JSON 方法风格兼容）：

```go
mmgin.SetMMResponse(c, http.StatusOK, data)
```

#### JSONC

直接返回 JSONC 格式的响应：

```go
mmgin.JSONC(c, http.StatusOK, data)
```

#### MetaMessage

直接返回 MetaMessage 二进制格式的响应：

```go
mmgin.MetaMessage(c, http.StatusOK, data)
```

#### AbortWithMetaMessage

发送 MetaMessage 格式的错误响应并终止请求：

```go
mmgin.AbortWithMetaMessage(c, http.StatusNotFound, ErrorResponse{
    Error: "user not found",
})
```

#### OptionsHandler

创建 OPTIONS 请求的处理器（Schema 发现）：

```go
mmgin.OPTIONS("/users", mmgin.OptionsHandler(CreateUserRequest{}))
```

---

### HTTP 客户端

`client` 包提供用于 MetaMessage 通信的泛型 HTTP 客户端。

#### Client

```go
c := client.NewClient("http://localhost:8080")
```

#### SetDefaultClient

设置全局默认客户端：

```go
client.SetDefaultClient("http://localhost:8080")
```

#### DoRequest

以类型安全的请求/响应执行泛型请求。POST/PUT/PATCH 会自动先发送 OPTIONS 预检请求验证 Schema：

```go
resp, err := client.DoRequest[CreateUserRequest, UserResponse](
    c, "POST", "/api/v1/users", req,
)
```

#### 便捷函数

使用默认客户端的包级别便捷函数：

```go
client.GET[any, ListUsersResponse]("/api/v1/users", nil)
client.POST[CreateUserRequest, UserResponse]("/api/v1/users", req)
client.PUT[UpdateUserRequest, APIResponse]("/api/v1/users/1", req)
client.DELETE[any, APIResponse]("/api/v1/users/1", nil)
client.PATCH[UpdateUserRequest, APIResponse]("/api/v1/users/1", req)
```

---

### Schema 发现

服务端的 POST/PUT/PATCH 路由会自动注册 OPTIONS 端点用于 Schema 发现。OPTIONS 响应会返回一个 MetaMessage 编码的结构体实例，其中包含完整的类型、约束条件与描述元数据。

客户端在发送实际请求前，会自动利用此机制进行请求验证：

```
客户端                          服务端
  │                               │
  ├── OPTIONS /api/v1/users ──────►
  │◄──── MetaMessage Schema ──────┤
  │     (struct definition)       │
  │                               │
  ├── POST /api/v1/users ────────►
  │◄──── MetaMessage Response ────┤
```

---

## 配置

### DecodeConfig

| 字段 | 类型 | 默认值 | 说明 |
|-------|------|---------|-------------|
| AllowJSONC | bool | true | 启用 JSONC 格式请求解析 |
| AllowMetaMessage | bool | true | 启用 MetaMessage 二进制格式请求解析 |
| DefaultFormat | FormatType | FormatAuto | 无法判断 Content-Type 时的默认解析格式 |
| MaxBodySize | int64 | 10MB | 最大请求体大小（0 = 不限） |

### EncodeConfig

| 字段 | 类型 | 默认值 | 说明 |
|-------|------|---------|-------------|
| DefaultFormat | FormatType | FormatMetaMessage | 默认响应编码格式 |
| AutoNegotiate | bool | false | 根据 Accept 请求头自动选择格式 |
| SuccessCode | int | 200 | 成功响应的 HTTP 状态码 |

### FormatType

```go
FormatAuto          // 自动检测
FormatJSONC         // JSONC 格式
FormatMetaMessage   // MetaMessage 二进制格式
```

### Content-Type 常量

```go
ContentTypeMetaMessage = "application/x-metamessage"
ContentTypeJSONC       = "application/jsonc"
```

---

## 示例

完整的服务端 + 客户端示例请见 [examples](examples/)。

```bash
cd examples
go run main.go
```

此示例展示：
- 使用 `Init()` 及泛型路由注册的服务端
- 使用 MetaMessage 二进制协议的 CRUD 操作
- 通过 OPTIONS 预检请求进行 Schema 验证的客户端
- 自定义验证及错误处理

---

## 依赖

- [gin-gonic/gin](https://github.com/gin-gonic/gin) - Web 框架
- [metamessage/metamessage](https://github.com/metamessage/metamessage) - MetaMessage 协议实现

## 许可证

MIT

---

**[中文](README.md)** | [English](README.en.md) | [日本語](README.ja.md) | [한국어](README.ko.md)