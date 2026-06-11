# mm-web-go

Go 웹 프레임워크에 MetaMessage 프로토콜 지원을 제공하며, 인코딩/디코딩, 데이터 바인딩, 스키마 발견 등의 기능을 포함합니다. Gin, Echo, Fiber, Chi, net/http를 지원합니다.

[中文](README.md) | [English](README.en.md) | [日本語](README.ja.md) | **한국어**

## 특징

- **한 줄 초기화**: `server.Init(r, "/api/v1")`으로 미들웨어와 라우트 그룹을 한 번에 등록
- **제네릭 라우트 등록**: `server.POST[T](path, handler)`로 자동 타입-safe 요청 바인딩, `GET`/`DELETE`도 지원
- **스키마 디스커버리**: 모든 제네릭 라우트가 자동으로 OPTIONS 엔드포인트를 등록하여 클라이언트 요청 검증
- **요청 디코딩**: JSONC 및 MetaMessage 바이너리 형식의 요청 본문 자동 감지 및 디코딩
- **응답 인코딩**: 응답 데이터를 MetaMessage 바이너리 형식으로 자동 인코딩
- **쿼리 파라미터 바인딩**: GET/DELETE는 `?data=<hex>` 쿼리 파라미터로 MetaMessage 데이터 바인딩
- **HTTP 클라이언트**: 스키마 사전 검증을 지원하는 제네릭 `DoRequest[REQ, RESP]`

## 설치

```bash
go get github.com/metamessage/mm-web-go
```

## 빠른 시작

### 서버

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

    // 한 줄 초기화: 미들웨어 + 라우트 그룹
    server.Init(r, "/api/v1")

    // 자동 바인딩 및 OPTIONS 스키마 디스커버리가 포함된 제네릭 라우트
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

### 클라이언트

```go
package main

import (
    "fmt"
    "github.com/metamessage/mm-web-go/client"
)

func main() {
    client.SetDefaultClient("http://localhost:8080", false)

    // 스키마 검증이 포함된 제네릭 요청
    req := CreateUserRequest{Name: "Alice", Email: "alice@example.com", Age: 25}
    resp, err := client.POST[CreateUserRequest, UserResponse]("/api/v1/users", &req)
    if err != nil {
        panic(err)
    }
    fmt.Printf("User: %+v\n", resp)
}
```

---

## 멀티 프레임워크 어댑터

모든 프레임워크가 **통일된 제네릭 라우트 등록 API**를 제공하며, 핸들러 시그니처는 `func(ctx, *T) (any, string, error)`입니다.

#### Gin

```go
import "github.com/gin-gonic/gin"
import server "github.com/metamessage/mm-web-go/mmgin"

r := gin.Default()
server.Init(r, "/api/v1")

// 제네릭 라우트: 자동 바인딩 + OPTIONS 스키마 디스커버리
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

> 비제네릭 라우트（`HEAD`/`OPTIONS`/`Any` 등）도 `server.HEAD()`、`server.OPTIONS()`、`server.Any()` 패키지 레벨 함수로 등록할 수 있습니다. 각 프레임워크의 네이티브 핸들러 타입을 사용합니다.

---

## API 문서

### 한 줄 초기화

#### Init

`Init`은 디코딩 및 인코딩 미들웨어를 등록하고, 라우트 그룹을 생성한 후 모든 후속 라우트 등록의 기본값으로 설정합니다.

```go
rg := mmgin.Init(r, "/api/v1")
// 이제 mmgin.GET/POST/PUT/DELETE 등이 모두 이 그룹을 사용합니다
```

---

### 라우트 등록

#### 제네릭 GET / DELETE

자동 쿼리 파라미터 바인딩과 OPTIONS 스키마 디스커버리가 포함된 제네릭 라우트 등록. `?data=<hex>` 쿼리 파라미터를 통해 MetaMessage 인코딩된 요청 데이터를 전달합니다.

```go
// Handler[T] 정의: func(c *gin.Context, req *T) (any, string, error)
type Handler[T any] func(c *gin.Context, req *T) (data any, tag string, err error)

mmgin.GET("/users", func(c *gin.Context, req *any) (any, string, error) {
    return ListUsersResponse{...}, "", nil
})

mmgin.DELETE("/users/:id", func(c *gin.Context, req *any) (any, string, error) {
    return APIResponse{...}, "", nil
})
```

#### 제네릭 POST / PUT / PATCH

자동 요청 본문 바인딩과 OPTIONS 스키마 디스커버리가 포함된 제네릭 라우트 등록:

```go
mmgin.POST("/users", func(c *gin.Context, req *CreateUserRequest) (any, string, error) {
    return UserResponse{...}, "", nil
})

mmgin.PUT("/users/:id", func(c *gin.Context, req *UpdateUserRequest) (any, string, error) {
    return APIResponse{...}, "", nil
})
```

각 제네릭 라우트는 동일한 경로에 OPTIONS 엔드포인트를 자동으로 등록합니다. 클라이언트는 OPTIONS 요청을 보내 요청 구조체 스키마(MetaMessage 바이너리로 인코딩됨)를 확인할 수 있습니다.

#### HEAD / OPTIONS / Any

자동 바인딩이 필요하지 않은 메서드의 표준 라우트 등록. 각 프레임워크의 네이티브 핸들러 타입 사용:

```go
mmgin.HEAD("/health", healthCheck)
mmgin.OPTIONS("/resources", optionsHandler)
mmgin.Any("/catch-all", catchAllHandler)
```

---

### 응답 함수

#### Respond

인코딩 미들웨어를 위한 응답 데이터를 설정합니다. 선택적 mm 태그 문자열을 받습니다:

```go
mmgin.Respond(c, User{Name: "Alice"}, "")
mmgin.Respond(c, users, "desc=User list response")
```

#### RespondWithStatus

커스텀 HTTP 상태 코드와 함께 응답 데이터를 설정합니다:

```go
mmgin.RespondWithStatus(c, http.StatusCreated, APIResponse{
    Code:    0,
    Message: "user created",
    Data:    &newUser,
}, "")
```

#### AbortWithMetaMessage

MetaMessage 형식의 오류 응답을 보내고 요청을 중단합니다:

```go
mmgin.AbortWithMetaMessage(c, http.StatusNotFound, ErrorResponse{
    Error: "user not found",
})
```

---

### 데이터 바인딩

#### Bind

요청 본문을 구조체에 바인딩합니다 (형식 자동 감지):

```go
var user User
if err := mmgin.Bind(c, &user); err != nil {
    // 오류 처리
}
```

#### BindQuery

쿼리 파라미터를 구조체에 바인딩합니다 (`?data=<hex>`에서 MetaMessage 인코딩 데이터 읽기):

```go
var filter Filter
if err := mmgin.BindQuery(c, &filter); err != nil {
    // 오류 처리
}
```

#### ShouldBind / ShouldBindWithTag

중단 대신 오류를 반환하는 비중단 변형:

```go
var user User
if err := mmgin.ShouldBind(c, &user); err != nil {
    // 수동으로 오류 처리
}
```

#### MustBindAndValidate

바인딩 및 검증 후, 실패 시 자동으로 오류 응답을 반환합니다:

```go
var req CreateUserRequest
if err := mmgin.MustBindAndValidate(c, &req); err != nil {
    return // 오류 응답이 이미 전송됨
}
```

#### Validator 인터페이스

커스텀 검증 로직을 위해 `Validator` 인터페이스를 구현합니다:

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

### HTTP 클라이언트

`client` 패키지는 MetaMessage 프로토콜 통신을 위한 제네릭 HTTP 클라이언트를 제공합니다.

#### Client

```go
c := client.NewClient("http://localhost:8080", false)
```

#### SetDefaultClient

전역 기본 클라이언트를 설정합니다:

```go
client.SetDefaultClient("http://localhost:8080", false)
```

#### DoRequest

타입-safe 요청/응답을 사용한 제네릭 요청 실행. 자동으로 OPTIONS 사전 요청을 보내 스키마를 검증합니다:

```go
resp, err := client.DoRequest[CreateUserRequest, UserResponse](
    c, "POST", "/api/v1/users", req,
)
```

#### 편의 함수

기본 클라이언트를 사용하는 패키지 레벨 편의 함수:

```go
client.GET[any, ListUsersResponse]("/api/v1/users", nil)
client.POST[CreateUserRequest, UserResponse]("/api/v1/users", req)
client.PUT[UpdateUserRequest, APIResponse]("/api/v1/users/1", req)
client.DELETE[any, APIResponse]("/api/v1/users/1", nil)
client.PATCH[UpdateUserRequest, APIResponse]("/api/v1/users/1", req)
```

---

### 스키마 디스커버리

서버의 제네릭 라우트(GET/POST/PUT/DELETE/PATCH)는 스키마 디스커버리를 위해 OPTIONS 엔드포인트를 자동으로 등록합니다. OPTIONS 응답은 전체 타입, 제약 조건 및 설명 메타데이터가 포함된 MetaMessage 인코딩 구조체 인스턴스를 반환합니다.

클라이언트는 실제 요청을 보내기 전에 이 메커니즘을 자동으로 사용하여 요청을 검증합니다:

```
클라이언트                       서버
  │                               │
  ├── OPTIONS /api/v1/users ──────►
  │◄──── MetaMessage 스키마 ──────┤
  │     (구조체 정의)              │
  │                               │
  ├── POST /api/v1/users ────────►
  │◄──── MetaMessage 응답 ────────┤
```

---

## 예제

완전한 서버 + 클라이언트 예제는 [examples](examples/)를 참조하세요.

```bash
cd examples/gin    # 또는 echo / fiber / chi / vanilla
go run main.go
```

이 예제는 다음을 보여줍니다:

- `Init()` 및 제네릭 라우트 등록을 사용한 서버
- GET/DELETE가 `?data=<hex>` 쿼리 파라미터로 요청 데이터를 전달하는 방법
- MetaMessage 바이너리 프로토콜을 사용한 CRUD 작업
- OPTIONS 사전 요청을 통한 스키마 검증을 사용하는 클라이언트

---

## 의존성

- [metamessage/metamessage](https://github.com/metamessage/metamessage) - MetaMessage 프로토콜 구현

## 라이선스

MIT

---

[中文](README.md) | [English](README.en.md) | [日本語](README.ja.md) | **한국어**
