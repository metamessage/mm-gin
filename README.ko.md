# mm-web-go

Go 웹 프레임워크에 MetaMessage 프로토콜 지원을 제공하며, 인코딩/디코딩, 데이터 바인딩, 스키마 발견 등의 기능을 포함합니다. Gin, Echo, Fiber, Vanilla를 지원합니다.

[中文](README.md) | [English](README.en.md) | [日本語](README.ja.md) | **한국어**

## 특징

- **한 줄 초기화**: `server.Init(r, "/api/v1")`으로 미들웨어와 라우트 그룹을 한 번에 등록
- **제네릭 라우트 등록**: `server.POST[T](path, handler)`로 자동 타입-safe 요청 바인딩
- **스키마 디스커버리**: POST/PUT/PATCH가 자동으로 OPTIONS를 등록하여 클라이언트 요청 검증
- **요청 디코딩**: JSONC 및 MetaMessage 바이너리 형식의 요청 본문 자동 감지 및 디코딩
- **응답 인코딩**: 응답 데이터를 MetaMessage 바이너리 형식으로 자동 인코딩
- **데이터 바인딩**: 요청 본문, 쿼리 파라미터, URI 파라미터, 헤더 바인딩 지원
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
    server.POST("/users", func(c *gin.Context, req *CreateUserRequest) {
        server.Respond(c, UserResponse{
            ID:    1,
            Name:  req.Name,
            Email: req.Email,
            Age:   req.Age,
        }, "")
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

### 멀티 프레임워크 어댑터

#### Gin

```go
import "github.com/gin-gonic/gin"
import server "github.com/metamessage/mm-web-go/mmgin"

r := gin.Default()
server.Init(r, "/api/v1")

// Gin 네이티브 API로 GET/DELETE 등 등록
r.GET("/users", listUsers)
r.GET("/users/:id", getUser)
r.DELETE("/users/:id", deleteUser)

// 통합 제네릭 API로 POST/PUT/PATCH 등록 (자동 바인딩 + OPTIONS 스키마 디스커버리)
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
import server "github.com/metamessage/mm-web-go/mmecho"

e := echo.New()
server.Init(e, "/api/v1")

// Echo 네이티브 API
e.GET("/users", listUsers)

// 동일 제네릭 핸들러, 프레임워크 비종속
server.POST("/users", func(r *http.Request, req *CreateUserRequest) (any, error) {
	return UserResponse{ID: 1, Name: req.Name, Email: req.Email, Age: req.Age}, nil
})
```

#### Fiber

```go
import "github.com/gofiber/fiber/v2"
import server "github.com/metamessage/mm-web-go/mmfiber"

app := fiber.New()
server.Init(app, "/api/v1")

// Fiber 네이티브 API
app.Get("/users", listUsers)

// 동일 제네릭 핸들러
server.POST("/users", func(r *http.Request, req *CreateUserRequest) (any, error) {
	return UserResponse{ID: 1, Name: req.Name, Email: req.Email, Age: req.Age}, nil
})
```

#### Chi

```go
import "github.com/go-chi/chi/v5"
import server "github.com/metamessage/mm-web-go/mmchi"

r := chi.NewRouter()
server.Init(r, "/api/v1")

// Chi 네이티브 API
r.Get("/users", listUsers)

// 동일 제네릭 핸들러
server.POST("/users", func(r *http.Request, req *CreateUserRequest) (any, error) {
	return UserResponse{ID: 1, Name: req.Name, Email: req.Email, Age: req.Age}, nil
})
```

#### net/http

```go
import server "github.com/metamessage/mm-web-go/mmvanilla"

mux := http.NewServeMux()
server.Init(mux, "/api/v1")

// 표준 라이브러리 네이티브 API
mux.HandleFunc("/api/v1/users", listUsers)

// 동일 제네릭 핸들러
server.POST("/users", func(r *http.Request, req *CreateUserRequest) (any, error) {
	return UserResponse{ID: 1, Name: req.Name, Email: req.Email, Age: req.Age}, nil
})
```

> 프레임워크 특화 `GET`, `HEAD`, `DELETE` 등의 라우트도 `server.GET()`, `server.DELETE()` 등의 패키지 레벨 함수로 등록할 수 있습니다. 활성 프레임워크의 네이티브 핸들러 타입을 받습니다.

```go
server.GET("/users", listUsers)
server.DELETE("/users/:id", deleteUser)
server.HEAD("/health", healthCheck)
server.OPTIONS("/resources", optionsHandler)
server.Any("/catch-all", catchAllHandler)
```

---

## API 문서

### 한 줄 초기화

#### Init

`Init`은 `MetaMessageDecoder`와 `MetaMessageEncoder` 미들웨어를 등록하고, 라우트 그룹을 생성한 후 모든 후속 라우트 등록의 기본값으로 설정합니다.

```go
rg := mmgin.Init(r, "/api/v1")
// 이제 mmgin.GET/POST/PUT/DELETE 등이 모두 이 그룹을 사용합니다
```

---

### 라우트 등록

#### GET / HEAD / DELETE / OPTIONS / Any

자동 바인딩이 필요하지 않은 메서드의 표준 라우트 등록:

```go
mmgin.GET("/users", listUsers)
mmgin.GET("/users/:id", getUser)
mmgin.DELETE("/users/:id", deleteUser)
mmgin.HEAD("/health", healthCheck)
mmgin.OPTIONS("/resources", optionsHandler)
mmgin.Any("/catch-all", catchAllHandler)
```

#### 제네릭 POST / PUT / PATCH

자동 요청 바인딩 및 OPTIONS 스키마 디스커버리가 포함된 제네릭 라우트 등록:

```go
// Handler[T any] 정의: func(c *gin.Context, req *T)
type Handler[T any] func(c *gin.Context, req *T)

mmgin.POST("/users", func(c *gin.Context, req *CreateUserRequest) {
    // req는 자동으로 바인딩 및 검증됩니다
    mmgin.Respond(c, UserResponse{...}, "")
})

mmgin.PUT("/users/:id", func(c *gin.Context, req *UpdateUserRequest) {
    mmgin.Respond(c, APIResponse{...}, "")
})
```

각 POST/PUT/PATCH 라우트는 동일한 경로에 OPTIONS 엔드포인트를 자동으로 등록합니다. 클라이언트는 OPTIONS 요청을 보내 요청 구조체 스키마(MetaMessage 바이너리로 인코딩됨)를 확인할 수 있습니다.

---

### 미들웨어

#### MetaMessageDecoder

JSONC 및 MetaMessage 바이너리 형식을 지원하는 요청 본문 디코딩 미들웨어.

```go
// 기본 설정 사용
r.Use(mmgin.MetaMessageDecoder(nil))

// 사용자 정의 설정
config := &mmgin.DecodeConfig{
    AllowJSONC:       true,
    AllowMetaMessage: true,
    DefaultFormat:    mmgin.FormatAuto,
    MaxBodySize:      10 << 20, // 10MB
}
r.Use(mmgin.MetaMessageDecoder(config))
```

#### MetaMessageEncoder

핸들러 응답 데이터를 MetaMessage 바이너리 형식으로 인코딩하는 응답 인코딩 미들웨어.

```go
// 기본 설정 사용
r.Use(mmgin.MetaMessageEncoder(nil))

// 사용자 정의 설정
config := &mmgin.EncodeConfig{
    DefaultFormat: mmgin.FormatMetaMessage,
    AutoNegotiate: false,
    SuccessCode:   http.StatusOK,
}
r.Use(mmgin.MetaMessageEncoder(config))
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

#### BindWithTag

지정된 mm 태그를 사용하여 요청 본문을 바인딩합니다:

```go
var user User
if err := mmgin.BindWithTag(c, &user, "desc=user"); err != nil {
    // 오류 처리
}
```

#### MustBind

바인딩하고, 실패 시 자동으로 400 오류 응답을 반환합니다:

```go
var user User
if err := mmgin.MustBind(c, &user); err != nil {
    return // 오류 응답이 이미 전송됨
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

#### BindQuery

쿼리 파라미터를 구조체에 바인딩합니다:

```go
var filter Filter
if err := mmgin.BindQuery(c, &filter); err != nil {
    // 오류 처리
}
```

#### BindHeader

요청 헤더를 구조체에 바인딩합니다:

```go
var headers Headers
if err := mmgin.BindHeader(c, &headers); err != nil {
    // 오류 처리
}
```

#### BindUri

URI 파라미터를 구조체에 바인딩합니다:

```go
var params Params
if err := mmgin.BindUri(c, &params); err != nil {
    // 오류 처리
}
```

#### AutoBind

모든 소스에서 자동으로 바인딩합니다 (우선순위: URI 파라미터 > 쿼리 파라미터 > 요청 본문):

```go
var req Request
if err := mmgin.AutoBind(c, &req); err != nil {
    // 오류 처리
}
```

---

### 데이터 검증

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

#### BindAndValidate

데이터를 바인딩하고 검증합니다:

```go
var req CreateUserRequest
if err := mmgin.BindAndValidate(c, &req); err != nil {
    // 오류 처리
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

#### SetMMResponse

응답 데이터를 직접 설정합니다 (gin의 JSON 메서드 스타일과 호환):

```go
mmgin.SetMMResponse(c, http.StatusOK, data)
```

#### JSONC

JSONC 형식의 응답을 직접 반환합니다:

```go
mmgin.JSONC(c, http.StatusOK, data)
```

#### MetaMessage

MetaMessage 바이너리 형식의 응답을 직접 반환합니다:

```go
mmgin.MetaMessage(c, http.StatusOK, data)
```

#### AbortWithMetaMessage

MetaMessage 형식의 오류 응답을 보내고 요청을 중단합니다:

```go
mmgin.AbortWithMetaMessage(c, http.StatusNotFound, ErrorResponse{
    Error: "user not found",
})
```

#### OptionsHandler

OPTIONS 요청용 핸들러를 생성합니다 (스키마 디스커버리):

```go
mmgin.OPTIONS("/users", mmgin.OptionsHandler(CreateUserRequest{}))
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

타입-safe 요청/응답을 사용한 제네릭 요청 실행. POST/PUT/PATCH의 경우 요청 스키마를 검증하기 위해 OPTIONS 사전 요청을 자동으로 보냅니다:

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

서버의 POST/PUT/PATCH 라우트는 스키마 디스커버리를 위해 OPTIONS 엔드포인트를 자동으로 등록합니다. OPTIONS 응답은 전체 타입, 제약 조건 및 설명 메타데이터가 포함된 MetaMessage 인코딩 구조체 인스턴스를 반환합니다.

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

## 설정

### DecodeConfig

| 필드 | 타입 | 기본값 | 설명 |
|-------|------|---------|-------------|
| AllowJSONC | bool | true | JSONC 형식 요청 파싱 활성화 |
| AllowMetaMessage | bool | true | MetaMessage 바이너리 형식 요청 파싱 활성화 |
| DefaultFormat | FormatType | FormatAuto | Content-Type을 확인할 수 없을 때의 기본 파싱 형식 |
| MaxBodySize | int64 | 10MB | 최대 요청 본문 크기 (0 = 제한 없음) |

### EncodeConfig

| 필드 | 타입 | 기본값 | 설명 |
|-------|------|---------|-------------|
| DefaultFormat | FormatType | FormatMetaMessage | 기본 응답 인코딩 형식 |
| AutoNegotiate | bool | false | Accept 헤더 기반 형식 자동 선택 |
| SuccessCode | int | 200 | 성공 응답의 HTTP 상태 코드 |

### FormatType

```go
FormatAuto          // 자동 감지
FormatJSONC         // JSONC 형식
FormatMetaMessage   // MetaMessage 바이너리 형식
```

### Content-Type 상수

```go
ContentTypeMetaMessage = "application/metamessage"
ContentTypeJSONC       = "application/jsonc"
```

---

## 예제

완전한 서버 + 클라이언트 예제는 [examples](examples/)를 참조하세요.

```bash
cd examples
go run main.go
```

이 예제는 다음을 보여줍니다:
- `Init()` 및 제네릭 라우트 등록을 사용한 서버
- MetaMessage 바이너리 프로토콜을 사용한 CRUD 작업
- OPTIONS 사전 요청을 통한 스키마 검증을 사용하는 클라이언트

---

## 의존성

- [metamessage/metamessage](https://github.com/metamessage/metamessage) - MetaMessage 프로토콜 구현

## 라이선스

MIT

---

[中文](README.md) | [English](README.en.md) | [日本語](README.ja.md) | **한국어**