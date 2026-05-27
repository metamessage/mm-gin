# mm-web-go

Go WebフレームワークにMetaMessageプロトコルのサポートを提供し、エンコード/デコード、データバインディング、スキーマ発見などの機能を備えています。Gin、Echo、Fiber、Vanillaに対応。

[中文](README.md) | [English](README.en.md) | **日本語** | [한국어](README.ko.md)

## 機能

- **一行初期化**: `server.Init(r, "/api/v1")` でミドルウェアとルートグループを一度に登録
- **ジェネリックルート登録**: `server.POST[T](path, handler)` による自動型安全リクエストバインディング
- **スキーマディスカバリー**: POST/PUT/PATCH が自動で OPTIONS を登録し、クライアントのリクエスト検証を支援
- **リクエストデコード**: JSONC および MetaMessage バイナリ形式のリクエストボディを自動検出・デコード
- **レスポンスエンコード**: レスポンスデータを MetaMessage バイナリ形式に自動エンコード
- **データバインディング**: リクエストボディ、クエリパラメータ、URI パラメータ、ヘッダーのバインディングをサポート
- **HTTP クライアント**: スキーマ事前検証付きジェネリック `DoRequest[REQ, RESP]`

## インストール

```bash
go get github.com/metamessage/mmgin
```

## クイックスタート

### サーバー

```go
package main

import (
    "github.com/gin-gonic/gin"
	server "github.com/metamessage/mmgin"
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

### クライアント

```go
package main

import (
    "fmt"
    "github.com/metamessage/client"
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

### マルチフレームワークアダプター

#### Gin

```go
import "github.com/gin-gonic/gin"
import server "github.com/metamessage/mmgin"

r := gin.Default()
server.Init(r, "/api/v1")

// Gin ネイティブ API で GET/DELETE などを登録
r.GET("/users", listUsers)
r.GET("/users/:id", getUser)
r.DELETE("/users/:id", deleteUser)

// 統一ジェネリック API で POST/PUT/PATCH を登録（自動バインド + OPTIONS スキーマディスカバリー）
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
import server "github.com/metamessage/mmecho"

e := echo.New()
server.Init(e, "/api/v1")

// Echo ネイティブ API
e.GET("/users", listUsers)

// 同一ジェネリックハンドラー、フレームワーク非依存
server.POST("/users", func(r *http.Request, req *CreateUserRequest) (any, error) {
	return UserResponse{ID: 1, Name: req.Name, Email: req.Email, Age: req.Age}, nil
})
```

#### Fiber

```go
import "github.com/gofiber/fiber/v2"
import server "github.com/metamessage/mmfiber"

app := fiber.New()
server.Init(app, "/api/v1")

// Fiber ネイティブ API
app.Get("/users", listUsers)

// 同一ジェネリックハンドラー
server.POST("/users", func(r *http.Request, req *CreateUserRequest) (any, error) {
	return UserResponse{ID: 1, Name: req.Name, Email: req.Email, Age: req.Age}, nil
})
```

#### Chi

```go
import "github.com/go-chi/chi/v5"
import server "github.com/metamessage/mmchi"

r := chi.NewRouter()
server.Init(r, "/api/v1")

// Chi ネイティブ API
r.Get("/users", listUsers)

// 同一ジェネリックハンドラー
server.POST("/users", func(r *http.Request, req *CreateUserRequest) (any, error) {
	return UserResponse{ID: 1, Name: req.Name, Email: req.Email, Age: req.Age}, nil
})
```

#### net/http

```go
import server "github.com/metamessage/mmvanilla"

mux := http.NewServeMux()
server.Init(mux, "/api/v1")

// 標準ライブラリネイティブ API
mux.HandleFunc("/api/v1/users", listUsers)

// 同一ジェネリックハンドラー
server.POST("/users", func(r *http.Request, req *CreateUserRequest) (any, error) {
	return UserResponse{ID: 1, Name: req.Name, Email: req.Email, Age: req.Age}, nil
})
```

> フレームワーク固有の `GET`、`HEAD`、`DELETE` などのルートも `server.GET()`、`server.DELETE()` などのパッケージレベル関数で登録可能です。アクティブなフレームワークのネイティブハンドラー型を受け付けます。

```go
server.GET("/users", listUsers)
server.DELETE("/users/:id", deleteUser)
server.HEAD("/health", healthCheck)
server.OPTIONS("/resources", optionsHandler)
server.Any("/catch-all", catchAllHandler)
```

---

## API ドキュメント

### 一行初期化

#### Init

`Init` は `MetaMessageDecoder` と `MetaMessageEncoder` ミドルウェアを登録し、ルートグループを作成して、以降のすべてのルート登録のデフォルトとして設定します。

```go
rg := mmgin.Init(r, "/api/v1")
// 以降 mmgin.GET/POST/PUT/DELETE などはすべてこのグループを使用
```

---

### ルート登録

#### GET / HEAD / DELETE / OPTIONS / Any

自動バインディングが不要なメソッドの標準ルート登録：

```go
mmgin.GET("/users", listUsers)
mmgin.GET("/users/:id", getUser)
mmgin.DELETE("/users/:id", deleteUser)
mmgin.HEAD("/health", healthCheck)
mmgin.OPTIONS("/resources", optionsHandler)
mmgin.Any("/catch-all", catchAllHandler)
```

#### ジェネリック POST / PUT / PATCH

自動リクエストバインディングと OPTIONS スキーマディスカバリーを備えたジェネリックルート登録：

```go
// Handler[T any] 定義: func(c *gin.Context, req *T)
type Handler[T any] func(c *gin.Context, req *T)

mmgin.POST("/users", func(c *gin.Context, req *CreateUserRequest) {
    // req は自動バインディング・検証済み
    mmgin.Respond(c, UserResponse{...}, "")
})

mmgin.PUT("/users/:id", func(c *gin.Context, req *UpdateUserRequest) {
    mmgin.Respond(c, APIResponse{...}, "")
})
```

各 POST/PUT/PATCH ルートは同じパスに OPTIONS エンドポイントを自動登録します。クライアントは OPTIONS リクエストを送信することで、リクエスト構造体のスキーマ（MetaMessage バイナリでエンコード）を取得できます。

---

### ミドルウェア

#### MetaMessageDecoder

JSONC および MetaMessage バイナリ形式をサポートするリクエストボディデコードミドルウェア。

```go
// デフォルト設定を使用
r.Use(mmgin.MetaMessageDecoder(nil))

// カスタム設定
config := &mmgin.DecodeConfig{
    AllowJSONC:       true,
    AllowMetaMessage: true,
    DefaultFormat:    mmgin.FormatAuto,
    MaxBodySize:      10 << 20, // 10MB
}
r.Use(mmgin.MetaMessageDecoder(config))
```

#### MetaMessageEncoder

ハンドラが設定したレスポンスデータを MetaMessage バイナリ形式にエンコードするレスポンスエンコードミドルウェア。

```go
// デフォルト設定を使用
r.Use(mmgin.MetaMessageEncoder(nil))

// カスタム設定
config := &mmgin.EncodeConfig{
    DefaultFormat: mmgin.FormatMetaMessage,
    AutoNegotiate: false,
    SuccessCode:   http.StatusOK,
}
r.Use(mmgin.MetaMessageEncoder(config))
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

#### BindWithTag

指定された mm タグを使用してリクエストボディをバインド：

```go
var user User
if err := mmgin.BindWithTag(c, &user, "custom_tag"); err != nil {
    // エラー処理
}
```

#### MustBind

バインドし、失敗時に自動で 400 エラーレスポンスを返す：

```go
var user User
if err := mmgin.MustBind(c, &user); err != nil {
    return // エラーレスポンスは自動送信済み
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

#### BindQuery

クエリパラメータを構造体にバインド：

```go
var filter Filter
if err := mmgin.BindQuery(c, &filter); err != nil {
    // エラー処理
}
```

#### BindHeader

リクエストヘッダーを構造体にバインド：

```go
var headers Headers
if err := mmgin.BindHeader(c, &headers); err != nil {
    // エラー処理
}
```

#### BindUri

URI パラメータを構造体にバインド：

```go
var params Params
if err := mmgin.BindUri(c, &params); err != nil {
    // エラー処理
}
```

#### AutoBind

すべてのソースから自動バインド（優先順位：URI パラメータ > クエリパラメータ > リクエストボディ）：

```go
var req Request
if err := mmgin.AutoBind(c, &req); err != nil {
    // エラー処理
}
```

---

### データ検証

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

#### BindAndValidate

データをバインドして検証：

```go
var req CreateUserRequest
if err := mmgin.BindAndValidate(c, &req); err != nil {
    // エラー処理
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

#### SetMMResponse

レスポンスデータを直接設定（gin の JSON メソッドスタイルと互換）：

```go
mmgin.SetMMResponse(c, http.StatusOK, data)
```

#### JSONC

JSONC 形式のレスポンスを直接返す：

```go
mmgin.JSONC(c, http.StatusOK, data)
```

#### MetaMessage

MetaMessage バイナリ形式のレスポンスを直接返す：

```go
mmgin.MetaMessage(c, http.StatusOK, data)
```

#### AbortWithMetaMessage

MetaMessage 形式のエラーレスポンスを送信してリクエストを中断：

```go
mmgin.AbortWithMetaMessage(c, http.StatusNotFound, ErrorResponse{
    Error: "user not found",
})
```

#### OptionsHandler

OPTIONS リクエスト用ハンドラを作成（スキーマディスカバリー）：

```go
mmgin.OPTIONS("/users", mmgin.OptionsHandler(CreateUserRequest{}))
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

型安全なリクエスト/レスポンスでジェネリックリクエストを実行。POST/PUT/PATCH は自動的に OPTIONS 事前リクエストを送信してスキーマを検証：

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

サーバーの POST/PUT/PATCH ルートはスキーマディスカバリー用に OPTIONS エンドポイントを自動登録します。OPTIONS レスポンスは、完全な型、制約、説明メタデータを含む MetaMessage エンコードされた構造体インスタンスを返します。

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

## 設定

### DecodeConfig

| フィールド | 型 | デフォルト | 説明 |
|-------|------|---------|-------------|
| AllowJSONC | bool | true | JSONC 形式のリクエスト解析を有効化 |
| AllowMetaMessage | bool | true | MetaMessage バイナリ形式のリクエスト解析を有効化 |
| DefaultFormat | FormatType | FormatAuto | Content-Type が不明な場合のデフォルト解析形式 |
| MaxBodySize | int64 | 10MB | 最大リクエストボディサイズ（0 = 制限なし） |

### EncodeConfig

| フィールド | 型 | デフォルト | 説明 |
|-------|------|---------|-------------|
| DefaultFormat | FormatType | FormatMetaMessage | デフォルトのレスポンスエンコード形式 |
| AutoNegotiate | bool | false | Accept ヘッダーに基づいて形式を自動選択 |
| SuccessCode | int | 200 | 成功レスポンスの HTTP ステータスコード |

### FormatType

```go
FormatAuto          // 自動検出
FormatJSONC         // JSONC 形式
FormatMetaMessage   // MetaMessage バイナリ形式
```

### Content-Type 定数

```go
ContentTypeMetaMessage = "application/x-metamessage"
ContentTypeJSONC       = "application/jsonc"
```

---

## 例

完全なサーバー + クライアントの例は [examples](examples/) を参照してください。

```bash
cd examples
go run main.go
```

この例では以下を紹介しています：
- `Init()` とジェネリックルート登録を使用したサーバー
- MetaMessage バイナリプロトコルを使用した CRUD 操作
- OPTIONS 事前リクエストによるスキーマ検証を使用するクライアント

---

## 依存関係

- [metamessage/metamessage](https://github.com/metamessage/metamessage) - MetaMessage プロトコル実装

## ライセンス

MIT

---

[中文](README.md) | [English](README.en.md) | **日本語** | [한국어](README.ko.md)