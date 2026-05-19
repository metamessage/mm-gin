# gin-mm

Gin framework plugin for MetaMessage, providing encoding/decoding and data binding functionality.

## Features

- **Request Decoding**: Automatically detect and decode JSONC and MetaMessage binary format request bodies
- **Response Encoding**: Automatically encode response data to JSONC or MetaMessage format
- **Data Binding**: Support for request body, query parameters, URI parameters, and header binding
- **Content Negotiation**: Automatically select response format based on Accept header
- **Custom Validation**: Support for struct custom validation logic

## Installation

```bash
go get github.com/metamessage/mm-gin
```

## Quick Start

```go
package main

import (
    "github.com/gin-gonic/gin"
    ginmm "github.com/metamessage/mm-gin"
)

type User struct {
    Name  string `mm:"type=str;desc=User name" json:"name"`
    Email string `mm:"type=email;desc=Email address" json:"email"`
    Age   int    `mm:"type=uint8;desc=Age" json:"age"`
}

func main() {
    r := gin.Default()

    // Use global middleware
    r.Use(ginmm.MetaMessageDecoder(nil))
    r.Use(ginmm.MetaMessageEncoder(nil))

    r.POST("/users", func(c *gin.Context) {
        var user User
        if err := ginmm.Bind(c, &user); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
        ginmm.Respond(c, user)
    })

    r.Run(":8080")
}
```

## API Documentation

### Middleware

#### MetaMessageDecoder

Request body decoding middleware, supports JSONC and MetaMessage binary formats.

```go
// Use default configuration
r.Use(ginmm.MetaMessageDecoder(nil))

// Custom configuration
config := &ginmm.DecodeConfig{
    AllowJSONC:       true,
    AllowMetaMessage: true,
    DefaultFormat:    ginmm.FormatAuto,
    MaxBodySize:      10 << 20, // 10MB
}
r.Use(ginmm.MetaMessageDecoder(config))
```

#### MetaMessageEncoder

Response encoding middleware, automatically encodes response data to specified format.

```go
// Use default configuration
r.Use(ginmm.MetaMessageEncoder(nil))

// Custom configuration
config := &ginmm.EncodeConfig{
    DefaultFormat: ginmm.FormatMetaMessage,
    AutoNegotiate: true,
    SuccessCode:   200,
}
r.Use(ginmm.MetaMessageEncoder(config))
```

### Data Binding

#### Bind

Bind request body to struct.

```go
var user User
if err := ginmm.Bind(c, &user); err != nil {
    // Handle error
}
```

#### MustBind

Bind data, automatically returns 400 error response on failure.

```go
var user User
if err := ginmm.MustBind(c, &user); err != nil {
    return // Error response already handled
}
```

#### BindQuery

Bind query parameters to struct.

```go
var filter Filter
if err := ginmm.BindQuery(c, &filter); err != nil {
    // Handle error
}
```

#### BindUri

Bind URI parameters to struct.

```go
var params Params
ginmm.BindUri(c, &params)
```

#### AutoBind

Automatically bind all data sources (URI params > Query params > Request body).

```go
var req Request
if err := ginmm.AutoBind(c, &req); err != nil {
    // Handle error
}
```

### Data Validation

#### Custom Validator

Implement the `Validator` interface:

```go
type CreateUserRequest struct {
    Name string `mm:"type=str" json:"name"`
    Age  int    `mm:"type=uint8" json:"age"`
}

func (r *CreateUserRequest) Validate() error {
    if r.Age < 18 {
        return errors.New("User must be at least 18 years old")
    }
    return nil
}
```

#### BindAndValidate

Bind and validate data:

```go
var req CreateUserRequest
if err := ginmm.BindAndValidate(c, &req); err != nil {
    // Handle error
}
```

#### MustBindAndValidate

Bind and validate, automatically returns error response on failure:

```go
var req CreateUserRequest
if err := ginmm.MustBindAndValidate(c, &req); err != nil {
    return // Error response already handled
}
```

### Response Functions

#### Respond

Set response data (handled by encoding middleware).

```go
ginmm.Respond(c, data)
```

#### RespondWithStatus

Set response data and status code.

```go
ginmm.RespondWithStatus(c, http.StatusCreated, data)
```

#### JSONC

Return JSONC format response directly.

```go
ginmm.JSONC(c, 200, data)
```

#### MetaMessage

Return MetaMessage binary format response directly.

```go
ginmm.MetaMessage(c, 200, data)
```

## Configuration Options

### DecodeConfig

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| AllowJSONC | bool | true | Allow JSONC format requests |
| AllowMetaMessage | bool | true | Allow MetaMessage binary format requests |
| DefaultFormat | FormatType | FormatAuto | Default parsing format |
| MaxBodySize | int64 | 10MB | Maximum request body size |

### EncodeConfig

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| DefaultFormat | FormatType | FormatMetaMessage | Default response format |
| AutoNegotiate | bool | true | Auto select format based on Accept header |
| SuccessCode | int | 200 | Success response status code |

### FormatType

```go
FormatAuto          // Auto detect
FormatJSONC         // JSONC format
FormatMetaMessage   // MetaMessage binary format
```

## Content-Type

- `application/x-metamessage` - MetaMessage binary format
- `application/jsonc` - JSONC format

## Examples

See the [examples](examples/) directory for complete examples.

```bash
cd examples
go run main.go
```

Test requests:

```bash
# Create user (JSONC)
curl -X POST http://localhost:8080/api/v1/users \
  -H 'Content-Type: application/jsonc' \
  -d '{"name":"Alice","email":"alice@example.com","age":25}'

# Get user list
curl http://localhost:8080/api/v1/users

# Get single user
curl http://localhost:8080/api/v1/users/1
```

## Dependencies

- [gin-gonic/gin](https://github.com/gin-gonic/gin) - Web framework
- [metamessage/metamessage](https://github.com/metamessage/metamessage) - MetaMessage protocol implementation

## License

MIT

---

**[中文](README.md)** | English | [日本語](README.ja.md)
