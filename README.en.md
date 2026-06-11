# mm-web-go

Provides MetaMessage protocol support for Go web frameworks, including encoding/decoding, data binding, schema discovery and other features. Supports Gin, Echo, Fiber, Chi, and net/http.

[中文](README.md) | **English** | [日本語](README.ja.md) | [한국어](README.ko.md)

## Features

- **One-line initialization**: `server.Init(r, "/api/v1")` registers middleware and route group at once
- **Generic route registration**: `server.POST[T](path, handler)` with automatic type-safe request binding, `GET`/`DELETE` also supported
- **Schema discovery**: All generic routes automatically register OPTIONS endpoints for client-side request validation
- **Request decoding**: Auto-detect and decode JSONC and MetaMessage binary request bodies
- **Response encoding**: Auto-encode response data into MetaMessage binary format
- **Query parameter binding**: GET/DELETE bind MetaMessage data via `?data=<hex>` query parameter
- **HTTP client**: Generic `DoRequest[REQ, RESP]` with schema preflight validation

## Installation

```bash
go get github.com/metamessage/mm-web-go
```

## Quick Start

### Server

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

    // One-line initialization: middleware + route group
    server.Init(r, "/api/v1")

    // Generic route with auto-binding and OPTIONS schema discovery
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

### Client

```go
package main

import (
    "fmt"
    "github.com/metamessage/mm-web-go/client"
)

func main() {
    client.SetDefaultClient("http://localhost:8080", false)

    // Generic request with schema validation
    req := CreateUserRequest{Name: "Alice", Email: "alice@example.com", Age: 25}
    resp, err := client.POST[CreateUserRequest, UserResponse]("/api/v1/users", &req)
    if err != nil {
        panic(err)
    }
    fmt.Printf("User: %+v\n", resp)
}
```

---

## Multi-Framework Adapter

All frameworks provide **a unified generic route registration API** with the handler signature `func(ctx, *T) (any, string, error)`.

#### Gin

```go
import "github.com/gin-gonic/gin"
import server "github.com/metamessage/mm-web-go/mmgin"

r := gin.Default()
server.Init(r, "/api/v1")

// Generic routes: auto-bind + OPTIONS schema discovery
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

#### net/http

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

> Non-generic routes (like `HEAD`/`OPTIONS`/`Any`) can also be registered via `server.HEAD()`, `server.OPTIONS()`, `server.Any()` package-level functions, using native handler types.

---

## API Documentation

### One-line Initialization

#### Init

`Init` registers decoding and encoding middleware, creates a route group, and sets it as the default for all subsequent route registrations.

```go
rg := mmgin.Init(r, "/api/v1")
// Now mmgin.GET/POST/PUT/DELETE/etc. all use this group
```

---

### Route Registration

#### Generic GET / DELETE

Generic route registration with automatic query parameter binding and OPTIONS schema discovery. Request data is passed via the `?data=<hex>` query parameter as MetaMessage-encoded data.

```go
// Handler[T] definition: func(c *gin.Context, req *T) (any, string, error)
type Handler[T any] func(c *gin.Context, req *T) (data any, tag string, err error)

mmgin.GET("/users", func(c *gin.Context, req *any) (any, string, error) {
    return ListUsersResponse{...}, "", nil
})

mmgin.DELETE("/users/:id", func(c *gin.Context, req *any) (any, string, error) {
    return APIResponse{...}, "", nil
})
```

#### Generic POST / PUT / PATCH

Generic route registration with automatic request body binding and OPTIONS schema discovery:

```go
mmgin.POST("/users", func(c *gin.Context, req *CreateUserRequest) (any, string, error) {
    return UserResponse{...}, "", nil
})

mmgin.PUT("/users/:id", func(c *gin.Context, req *UpdateUserRequest) (any, string, error) {
    return APIResponse{...}, "", nil
})
```

Each generic route automatically registers an OPTIONS endpoint on the same path. Clients can send OPTIONS requests to discover the request struct schema (encoded as MetaMessage binary).

#### HEAD / OPTIONS / Any

Standard route registration for methods without auto-binding, using native handler types:

```go
mmgin.HEAD("/health", healthCheck)
mmgin.OPTIONS("/resources", optionsHandler)
mmgin.Any("/catch-all", catchAllHandler)
```

---

### Response Functions

#### Respond

Set response data for the encoding middleware. Accepts an optional mm tag string:

```go
mmgin.Respond(c, User{Name: "Alice"}, "")
mmgin.Respond(c, users, "desc=User list response")
```

#### RespondWithStatus

Set response data with a custom HTTP status code:

```go
mmgin.RespondWithStatus(c, http.StatusCreated, APIResponse{
    Code:    0,
    Message: "user created",
    Data:    &newUser,
}, "")
```

#### AbortWithMetaMessage

Send a MetaMessage-format error response and abort the request:

```go
mmgin.AbortWithMetaMessage(c, http.StatusNotFound, ErrorResponse{
    Error: "user not found",
})
```

---

### Data Binding

#### Bind

Bind request body to a struct (auto-detects format):

```go
var user User
if err := mmgin.Bind(c, &user); err != nil {
    // Handle error
}
```

#### BindQuery

Bind query parameters to a struct (reads MetaMessage-encoded data from `?data=<hex>`):

```go
var filter Filter
if err := mmgin.BindQuery(c, &filter); err != nil {
    // Handle error
}
```

#### ShouldBind / ShouldBindWithTag

Non-panicking variants that return errors instead of aborting:

```go
var user User
if err := mmgin.ShouldBind(c, &user); err != nil {
    // Handle error manually
}
```

#### MustBindAndValidate

Bind and validate, automatically returning an error response on failure:

```go
var req CreateUserRequest
if err := mmgin.MustBindAndValidate(c, &req); err != nil {
    return // Error response already sent
}
```

#### Validator Interface

Implement the `Validator` interface for custom validation logic:

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

### HTTP Client

The `client` package provides a generic HTTP client for MetaMessage protocol communication.

#### Client

```go
c := client.NewClient("http://localhost:8080", false)
```

#### SetDefaultClient

Set a global default client:

```go
client.SetDefaultClient("http://localhost:8080", false)
```

#### DoRequest

Generic request execution with type-safe request/response. Automatically sends an OPTIONS preflight to validate the request schema:

```go
resp, err := client.DoRequest[CreateUserRequest, UserResponse](
    c, "POST", "/api/v1/users", req,
)
```

#### Convenience Functions

Package-level convenience functions using the default client:

```go
client.GET[any, ListUsersResponse]("/api/v1/users", nil)
client.POST[CreateUserRequest, UserResponse]("/api/v1/users", req)
client.PUT[UpdateUserRequest, APIResponse]("/api/v1/users/1", req)
client.DELETE[any, APIResponse]("/api/v1/users/1", nil)
client.PATCH[UpdateUserRequest, APIResponse]("/api/v1/users/1", req)
```

---

### Schema Discovery

Generic routes (GET/POST/PUT/DELETE/PATCH) on the server automatically register an OPTIONS endpoint for schema discovery. The OPTIONS response returns a MetaMessage-encoded struct instance containing full type, constraint, and description metadata.

The client automatically uses this mechanism for request validation before sending the actual request:

```
Client                          Server
  │                               │
  ├── OPTIONS /api/v1/users ──────►
  │◄──── MetaMessage Schema ──────┤
  │     (struct definition)       │
  │                               │
  ├── POST /api/v1/users ────────►
  │◄──── MetaMessage Response ────┤
```

---

## Examples

See [examples](examples/) for a complete server + client example.

```bash
cd examples/gin    # or echo / fiber / chi / vanilla
go run main.go
```

The example demonstrates:
- Server with `Init()` and generic route registration
- GET/DELETE passing request data via `?data=<hex>` query parameter
- CRUD operations with MetaMessage binary protocol
- Client with schema validation via OPTIONS preflight

---

## Dependencies

- [metamessage/metamessage](https://github.com/metamessage/metamessage) - MetaMessage protocol implementation

## License

MIT

---

[中文](README.md) | **English** | [日本語](README.ja.md) | [한국어](README.ko.md)