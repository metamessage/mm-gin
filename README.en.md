# mm-gin

Gin framework plugin for MetaMessage protocol, providing encoding/decoding, data binding, schema discovery and generic route registration.

[中文](README.md) | **English** | [日本語](README.ja.md) | [한국어](README.ko.md)

## Features

- **One-line initialization**: `mmgin.Init(r, "/api/v1")` registers middleware and route group at once
- **Generic route registration**: `mmgin.POST[T](path, handler)` with automatic type-safe request binding
- **Schema discovery**: POST/PUT/PATCH automatically register OPTIONS for client-side request validation
- **Request decoding**: Auto-detect and decode JSONC and MetaMessage binary request bodies
- **Response encoding**: Auto-encode response data into MetaMessage binary format
- **Data binding**: Support for request body, query parameters, URI parameters, and headers
- **Custom validation**: Support for struct-level validation via `Validator` interface
- **HTTP client**: Generic `DoRequest[REQ, RESP]` with schema preflight validation

## Installation

```bash
go get github.com/metamessage/mm-gin
```

## Quick Start

### Server

```go
package main

import (
    "github.com/gin-gonic/gin"
    mmgin "github.com/metamessage/mm-gin"
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
    mmgin.Init(r, "/api/v1")

    // Generic route with auto-binding and OPTIONS schema discovery
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

### Client

```go
package main

import (
    "fmt"
    "github.com/metamessage/mm-gin/client"
)

func main() {
    client.SetDefaultClient("http://localhost:8080")

    // Generic request with schema validation
    req := CreateUserRequest{Name: "Alice", Email: "alice@example.com", Age: 25}
    resp, err := client.POST[CreateUserRequest, UserResponse]("/api/v1/users", req)
    if err != nil {
        panic(err)
    }
    fmt.Printf("User: %+v\n", resp)
}
```

## API Documentation

### One-line Initialization

#### Init

`Init` registers `MetaMessageDecoder` and `MetaMessageEncoder` middleware, creates a route group, and sets it as the default for all subsequent route registrations.

```go
rg := mmgin.Init(r, "/api/v1")
// Now mmgin.GET/POST/PUT/DELETE/etc. all use this group
```

---

### Route Registration

#### GET / HEAD / DELETE / OPTIONS / Any

Standard route registration for methods without auto-binding:

```go
mmgin.GET("/users", listUsers)
mmgin.GET("/users/:id", getUser)
mmgin.DELETE("/users/:id", deleteUser)
mmgin.HEAD("/health", healthCheck)
mmgin.OPTIONS("/resources", optionsHandler)
mmgin.Any("/catch-all", catchAllHandler)
```

#### Generic POST / PUT / PATCH

Generic route registration with automatic request binding and OPTIONS schema discovery:

```go
// Handler[T any] definition: func(c *gin.Context, req *T)
type Handler[T any] func(c *gin.Context, req *T)

mmgin.POST("/users", func(c *gin.Context, req *CreateUserRequest) {
    // req is auto-bound and validated
    mmgin.Respond(c, UserResponse{...}, "")
})

mmgin.PUT("/users/:id", func(c *gin.Context, req *UpdateUserRequest) {
    mmgin.Respond(c, APIResponse{...}, "")
})
```

Each POST/PUT/PATCH route automatically registers an OPTIONS endpoint on the same path. Clients can send OPTIONS requests to discover the request struct schema (encoded as MetaMessage binary).

---

### Middleware

#### MetaMessageDecoder

Request body decoding middleware supporting JSONC and MetaMessage binary formats.

```go
// Default configuration
r.Use(mmgin.MetaMessageDecoder(nil))

// Custom configuration
config := &mmgin.DecodeConfig{
    AllowJSONC:       true,
    AllowMetaMessage: true,
    DefaultFormat:    mmgin.FormatAuto,
    MaxBodySize:      10 << 20, // 10MB
}
r.Use(mmgin.MetaMessageDecoder(config))
```

#### MetaMessageEncoder

Response encoding middleware that encodes handler response data into MetaMessage binary format.

```go
// Default configuration
r.Use(mmgin.MetaMessageEncoder(nil))

// Custom configuration
config := &mmgin.EncodeConfig{
    DefaultFormat: mmgin.FormatMetaMessage,
    AutoNegotiate: false,
    SuccessCode:   http.StatusOK,
}
r.Use(mmgin.MetaMessageEncoder(config))
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

#### BindWithTag

Bind request body using a specified mm tag:

```go
var user User
if err := mmgin.BindWithTag(c, &user, "custom_tag"); err != nil {
    // Handle error
}
```

#### MustBind

Bind and automatically return a 400 error response on failure:

```go
var user User
if err := mmgin.MustBind(c, &user); err != nil {
    return // Error response already sent
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

#### BindQuery

Bind query parameters to a struct:

```go
var filter Filter
if err := mmgin.BindQuery(c, &filter); err != nil {
    // Handle error
}
```

#### BindHeader

Bind request headers to a struct:

```go
var headers Headers
if err := mmgin.BindHeader(c, &headers); err != nil {
    // Handle error
}
```

#### BindUri

Bind URI parameters to a struct:

```go
var params Params
if err := mmgin.BindUri(c, &params); err != nil {
    // Handle error
}
```

#### AutoBind

Automatically bind from all sources with priority: URI params > query params > request body:

```go
var req Request
if err := mmgin.AutoBind(c, &req); err != nil {
    // Handle error
}
```

---

### Data Validation

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

#### BindAndValidate

Bind and validate data:

```go
var req CreateUserRequest
if err := mmgin.BindAndValidate(c, &req); err != nil {
    // Handle error
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

#### SetMMResponse

Set response data directly (compatible with gin's JSON method style):

```go
mmgin.SetMMResponse(c, http.StatusOK, data)
```

#### JSONC

Return a JSONC-format response directly:

```go
mmgin.JSONC(c, http.StatusOK, data)
```

#### MetaMessage

Return a MetaMessage binary-format response directly:

```go
mmgin.MetaMessage(c, http.StatusOK, data)
```

#### AbortWithMetaMessage

Send a MetaMessage-format error response and abort the request:

```go
mmgin.AbortWithMetaMessage(c, http.StatusNotFound, ErrorResponse{
    Error: "user not found",
})
```

#### OptionsHandler

Create a handler for OPTIONS requests (schema discovery):

```go
mmgin.OPTIONS("/users", mmgin.OptionsHandler(CreateUserRequest{}))
```

---

### HTTP Client

The `client` package provides a generic HTTP client for MetaMessage protocol communication.

#### Client

```go
c := client.NewClient("http://localhost:8080")
```

#### SetDefaultClient

Set a global default client:

```go
client.SetDefaultClient("http://localhost:8080")
```

#### DoRequest

Generic request execution with type-safe request/response. For POST/PUT/PATCH, automatically sends an OPTIONS preflight to validate the request schema:

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

POST/PUT/PATCH routes on the server automatically register an OPTIONS endpoint for schema discovery. The OPTIONS response returns a MetaMessage-encoded struct instance containing full type, constraint, and description metadata.

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

## Configuration

### DecodeConfig

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| AllowJSONC | bool | true | Enable JSONC format request parsing |
| AllowMetaMessage | bool | true | Enable MetaMessage binary format request parsing |
| DefaultFormat | FormatType | FormatAuto | Default parsing format when Content-Type is unavailable |
| MaxBodySize | int64 | 10MB | Maximum request body size (0 = unlimited) |

### EncodeConfig

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| DefaultFormat | FormatType | FormatMetaMessage | Default response encoding format |
| AutoNegotiate | bool | false | Auto-select format based on Accept header |
| SuccessCode | int | 200 | HTTP status code for successful responses |

### FormatType

```go
FormatAuto          // Auto-detect
FormatJSONC         // JSONC format
FormatMetaMessage   // MetaMessage binary format
```

### Content-Type Constants

```go
ContentTypeMetaMessage = "application/x-metamessage"
ContentTypeJSONC       = "application/jsonc"
```

---

## Examples

See [examples](examples/) for a complete server + client example.

```bash
cd examples
go run main.go
```

The example demonstrates:
- Server with `Init()` and generic route registration
- CRUD operations with MetaMessage binary protocol
- Client with schema validation via OPTIONS preflight
- Custom validation and error handling

---

## Dependencies

- [gin-gonic/gin](https://github.com/gin-gonic/gin) - Web framework
- [metamessage/metamessage](https://github.com/metamessage/metamessage) - MetaMessage protocol implementation

## License

MIT

---

[中文](README.md) | **English** | [日本語](README.ja.md) | [한국어](README.ko.md)