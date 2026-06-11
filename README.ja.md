# mm-web-go

Go WebフレームワークにMetaMessageプロトコルのサポートを提供し、エンコード/デコード、データバインディング、スキーマ発見などの機能を備えています。Gin、Echo、Fiber、Chi、net/httpに対応。

[中文](README.md) | [English](README.en.md) | **日本語** | [한국어](README.ko.md)

## 機能

- **一行初期化**: `server.Init(r, "/api/v1")` でミドルウェアとルートグループを一度に登録
- **ジェネリックルート登録**: `server.POST[T](path, handler)` による自動型安全リクエストバインディング、`GET`/`DELETE` も対応
- **スキーマディスカバリー**: すべてのジェネリックルートが自動で OPTIONS エンドポイントを登録し、クライアントのリクエスト検証を支援
- **リクエストデコード**: JSONC および MetaMessage バイナリ形式のリクエストボディを自動検出・デコード
- **レスポンスエンコード**: レスポンスデータを MetaMessage バイナリ形式に自動エンコード
- **クエリパラメータバインディング**: GET/DELETE は `?data=<hex>` クエリパラメータ経由で MetaMessage データをバインド
- **HTTP クライアント**: スキーマ事前検証付きジェネリック `DoRequest[REQ, RESP]`

## インストール

```bash
go get github.com/metamessage/mm-web-go
```

## クイックスタート

### サーバー

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

    // 一行初期化：ミドルウェア + ルートグループ
    server.Init(r, "/api/v1")

    // 自動バインディングと OPTIONS スキーマディスカバリー付きジェネリックルート
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

### クライアント

```go
package main

import (
    "fmt"
    "github.com/metamessage/mm-web-go/client"
)

func main() {
    client.SetDefaultClient("http://localhost:8080", false)

    // スキーマ検証付きジェネリックリクエスト
    req := CreateUserRequest{Name: "Alice", Email: "alice@example.com", Age: 25}
    resp, err := client.POST[CreateUserRequest, UserResponse]("/api/v1/users", &req)
    if err != nil {
        panic(err)
    }
    fmt.Printf("User: %+v\n", resp)
}
```

---

## マルチフレームワークアダプター

すべてのフレームワークは**統一されたジェネリックルート登録 API**を提供し、ハンドラーのシグネチャは `func(ctx, *T) (any, string, error)` です。

#### Gin

```go
import "github.com/gin-gonic/gin"
import server "github.com/metamessage/mm-web-go/mmgin"

r := gin.Default()
server.Init(r, "/api/v1")

// ジェネリックルート：自動バインド + OPTIONS スキーマディスカバリー
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

> 非ジェネリックルート（`HEAD`/`OPTIONS`/`Any` など）も `server.HEAD()`、`server.OPTIONS()`、`server.Any()` パッケージレベル関数で登録可能です。各フレームワークのネイティブハンドラー型を使用します。

---

## API ドキュメント

### 一行初期化

#### Init

`Init` はデコードとエンコードのミドルウェアを登録し、ルートグループを作成して、以降のすべてのルート登録のデフォルトとして設定します。

```go
rg := mmgin.Init(r, "/api/v1")
// 以降 mmgin.GET/POST/PUT/DELETE などはすべてこのグループを使用
```

---

### ルート登録

#### ジェネリック GET / DELETE

自動クエリパラメータバインディングと OPTIONS スキーマディスカバリーを備えたジェネリックルート登録。リクエストデータは `?data=<hex>` クエリパラメータ経由で MetaMessage エンコードされて渡されます。

```go
// Handler[T] 定義: func(c *gin.Context, req *T) (any, string, error)
type Handler[T any] func(c *gin.Context, req *T) (data any, tag string, err error)

mmgin.GET("/users", func(c *gin.Context, req *any) (any, string, error) {
    return ListUsersResponse{...}, "", nil
})

mmgin.DELETE("/users/:id", func(c *gin.Context, req *any) (any, string, error) {
    return APIResponse{...}, "", nil
})
```

#### ジェネリック POST / PUT / PATCH

自動リクエストボディバインディングと OPTIONS スキーマディスカバリーを備えたジェネリックルート登録：

```go
mmgin.POST("/users", func(c *gin.Context, req *CreateUserRequest) (any, string, error) {
    return UserResponse{...}, "", nil
})

mmgin.PUT("/users/:id", func(c *gin.Context, req *UpdateUserRequest) (any, string, error) {
    return APIResponse{...}, "", nil
})
```

各ジェネリックルートは同じパスに OPTIONS エンドポイントを自動登録します。クライアントは OPTIONS リクエストを送信することで、リクエスト構造体のスキーマ（MetaMessage バイナリでエンコード）を取得できます。

#### HEAD / OPTIONS / Any

自動バインディングが不要なメソッドの標準ルート登録。各フレームワークのネイティブハンドラー型を使用：

```go
mmgin.HEAD("/health", healthCheck)
mmgin.OPTIONS("/resources", optionsHandler)
mmgin.Any("/catch-all", catchAllHandler)
```

---

### レスポンス関数

#### Respond

エンコードミドルウェア用にレスポンスデータを設定。オプションの mm タグ文字列を受け付けます：

```go
mmgin.Respond(c, User{Name: "Alice"}, "")
mmgin.Respond(c, users, "desc=User list response")
```

#### RespondWithStatus

カスタム HTTP ステータスコードとともにレスポンスデータを設定：

```go
mmgin.RespondWithStatus(c, http.StatusCreated, APIResponse{
    Code:    0,
    Message: "user created",
    Data:    &newUser,
}, "")
```

#### AbortWithMetaMessage

MetaMessage 形式のエラーレスポンスを送信してリクエストを中断：

```go
mmgin.AbortWithMetaMessage(c, http.StatusNotFound, ErrorResponse{
    Error: "user not found",
})
```

---

### データバインディング

#### Bind

リクエストボディを構造体にバインド（形式自動検出）：

```go
var user User
if err := mmgin.Bind(c, &user); err != nil {
    // エラー処理
}
```

#### BindQuery

クエリパラメータを構造体にバインド（`?data=<hex>` から MetaMessage エンコードデータを読み取り）：

```go
var filter Filter
if err := mmgin.BindQuery(c, &filter); err != nil {
    // エラー処理
}
```

#### ShouldBind / ShouldBindWithTag

中断せずにエラーを返す亜種：

```go
var user User
if err := mmgin.ShouldBind(c, &user); err != nil {
    // 手動でエラー処理
}
```

#### MustBindAndValidate

バインドと検証を行い、失敗時に自動でエラーレスポンスを返す：

```go
var req CreateUserRequest
if err := mmgin.MustBindAndValidate(c, &req); err != nil {
    return // エラーレスポンスは自動送信済み
}
```

#### Validator インターフェース

カスタム検証ロジックのために `Validator` インターフェースを実装：

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

### HTTP クライアント

`client` パッケージは MetaMessage プロトコル通信のためのジェネリック HTTP クライアントを提供します。

#### Client

```go
c := client.NewClient("http://localhost:8080", false)
```

#### SetDefaultClient

グローバルデフォルトクライアントを設定：

```go
client.SetDefaultClient("http://localhost:8080", false)
```

#### DoRequest

型安全なリクエスト/レスポンスでジェネリックリクエストを実行。自動的に OPTIONS 事前リクエストを送信してスキーマを検証：

```go
resp, err := client.DoRequest[CreateUserRequest, UserResponse](
    c, "POST", "/api/v1/users", req,
)
```

#### 便利関数

デフォルトクライアントを使用するパッケージレベルの便利関数：

```go
client.GET[any, ListUsersResponse]("/api/v1/users", nil)
client.POST[CreateUserRequest, UserResponse]("/api/v1/users", req)
client.PUT[UpdateUserRequest, APIResponse]("/api/v1/users/1", req)
client.DELETE[any, APIResponse]("/api/v1/users/1", nil)
client.PATCH[UpdateUserRequest, APIResponse]("/api/v1/users/1", req)
```

---

### スキーマディスカバリー

サーバーのジェネリックルート（GET/POST/PUT/DELETE/PATCH）はスキーマディスカバリー用に OPTIONS エンドポイントを自動登録します。OPTIONS レスポンスは、完全な型、制約、説明メタデータを含む MetaMessage エンコードされた構造体インスタンスを返します。

クライアントは実際のリクエストを送信する前に、このメカニズムを自動的に使用してリクエストを検証します：

```
クライアント                        サーバー
  │                               │
  ├── OPTIONS /api/v1/users ──────►
  │◄──── MetaMessage スキーマ ─────┤
  │     (struct definition)       │
  │                               │
  ├── POST /api/v1/users ────────►
  │◄──── MetaMessage レスポンス ───┤
```

---

## 例

完全なサーバー + クライアントの例は [examples](examples/) を参照してください。

```bash
cd examples/gin    # または echo / fiber / chi / vanilla
go run main.go
```

この例では以下を紹介しています：
- `Init()` とジェネリックルート登録を使用したサーバー
- GET/DELETE が `?data=<hex>` クエリパラメータ経由でリクエストデータを渡す方法
- MetaMessage バイナリプロトコルを使用した CRUD 操作
- OPTIONS 事前リクエストによるスキーマ検証を使用するクライアント

---

## 依存関係

- [metamessage/metamessage](https://github.com/metamessage/metamessage) - MetaMessage プロトコル実装

## ライセンス

MIT

---

[中文](README.md) | [English](README.en.md) | **日本語** | [한국어](README.ko.md)