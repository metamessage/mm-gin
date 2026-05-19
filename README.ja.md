# gin-mm

MetaMessage の Gin フレームワークプラグイン。エンコード/デコードとデータバインディング機能を提供します。

## 機能

- **リクエストデコード**: JSONC と MetaMessage バイナリ形式のリクエストボディを自動検出・デコード
- **レスポンスエンコード**: レスポンスデータを JSONC または MetaMessage 形式に自動エンコード
- **データバインディング**: リクエストボディ、クエリパラメータ、URI パラメータ、ヘッダーのバインディングをサポート
- **コンテンツネゴシエーション**: Accept ヘッダーに基づいてレスポンス形式を自動選択
- **カスタム検証**: 構造体のカスタム検証ロジックをサポート

## インストール

```bash
go get github.com/metamessage/mm-gin
```

## クイックスタート

```go
package main

import (
    "github.com/gin-gonic/gin"
    ginmm "github.com/metamessage/mm-gin"
)

type User struct {
    Name  string `mm:"type=str;desc=ユーザー名" json:"name"`
    Email string `mm:"type=email;desc=メールアドレス" json:"email"`
    Age   int    `mm:"type=uint8;desc=年齢" json:"age"`
}

func main() {
    r := gin.Default()

    // グローバルミドルウェアを使用
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

## API ドキュメント

### ミドルウェア

#### MetaMessageDecoder

リクエストボディデコードミドルウェア。JSONC と MetaMessage バイナリ形式をサポート。

```go
// デフォルト設定を使用
r.Use(ginmm.MetaMessageDecoder(nil))

// カスタム設定
config := &ginmm.DecodeConfig{
    AllowJSONC:       true,
    AllowMetaMessage: true,
    DefaultFormat:    ginmm.FormatAuto,
    MaxBodySize:      10 << 20, // 10MB
}
r.Use(ginmm.MetaMessageDecoder(config))
```

#### MetaMessageEncoder

レスポンスエンコードミドルウェア。レスポンスデータを指定形式に自動エンコード。

```go
// デフォルト設定を使用
r.Use(ginmm.MetaMessageEncoder(nil))

// カスタム設定
config := &ginmm.EncodeConfig{
    DefaultFormat: ginmm.FormatMetaMessage,
    AutoNegotiate: true,
    SuccessCode:   200,
}
r.Use(ginmm.MetaMessageEncoder(config))
```

### データバインディング

#### Bind

リクエストボディを構造体にバインド。

```go
var user User
if err := ginmm.Bind(c, &user); err != nil {
    // エラー処理
}
```

#### MustBind

データをバインド。失敗時は自動的に 400 エラーレスポンスを返す。

```go
var user User
if err := ginmm.MustBind(c, &user); err != nil {
    return // エラーレスポンスは既に処理済み
}
```

#### BindQuery

クエリパラメータを構造体にバインド。

```go
var filter Filter
if err := ginmm.BindQuery(c, &filter); err != nil {
    // エラー処理
}
```

#### BindUri

URI パラメータを構造体にバインド。

```go
var params Params
ginmm.BindUri(c, &params)
```

#### AutoBind

すべてのデータソースを自動バインド（URI パラメータ > クエリパラメータ > リクエストボディ）。

```go
var req Request
if err := ginmm.AutoBind(c, &req); err != nil {
    // エラー処理
}
```

### データ検証

#### カスタムバリデータ

`Validator` インターフェースを実装：

```go
type CreateUserRequest struct {
    Name string `mm:"type=str" json:"name"`
    Age  int    `mm:"type=uint8" json:"age"`
}

func (r *CreateUserRequest) Validate() error {
    if r.Age < 18 {
        return errors.New("ユーザーは18歳以上である必要があります")
    }
    return nil
}
```

#### BindAndValidate

データをバインドして検証：

```go
var req CreateUserRequest
if err := ginmm.BindAndValidate(c, &req); err != nil {
    // エラー処理
}
```

#### MustBindAndValidate

バインドと検証を行い、失敗時は自動的にエラーレスポンスを返す：

```go
var req CreateUserRequest
if err := ginmm.MustBindAndValidate(c, &req); err != nil {
    return // エラーレスポンスは既に処理済み
}
```

### レスポンス関数

#### Respond

レスポンスデータを設定（エンコードミドルウェアによって処理）。

```go
ginmm.Respond(c, data)
```

#### RespondWithStatus

レスポンスデータとステータスコードを設定。

```go
ginmm.RespondWithStatus(c, http.StatusCreated, data)
```

#### JSONC

JSONC 形式のレスポンスを直接返す。

```go
ginmm.JSONC(c, 200, data)
```

#### MetaMessage

MetaMessage バイナリ形式のレスポンスを直接返す。

```go
ginmm.MetaMessage(c, 200, data)
```

## 設定オプション

### DecodeConfig

| フィールド | 型 | デフォルト | 説明 |
|-------|------|---------|-------------|
| AllowJSONC | bool | true | JSONC 形式のリクエストを許可 |
| AllowMetaMessage | bool | true | MetaMessage バイナリ形式のリクエストを許可 |
| DefaultFormat | FormatType | FormatAuto | デフォルトの解析形式 |
| MaxBodySize | int64 | 10MB | 最大リクエストボディサイズ |

### EncodeConfig

| フィールド | 型 | デフォルト | 説明 |
|-------|------|---------|-------------|
| DefaultFormat | FormatType | FormatMetaMessage | デフォルトのレスポンス形式 |
| AutoNegotiate | bool | true | Accept ヘッダーに基づいて形式を自動選択 |
| SuccessCode | int | 200 | 成功レスポンスのステータスコード |

### FormatType

```go
FormatAuto          // 自動検出
FormatJSONC         // JSONC 形式
FormatMetaMessage   // MetaMessage バイナリ形式
```

## Content-Type

- `application/x-metamessage` - MetaMessage バイナリ形式
- `application/jsonc` - JSONC 形式

## 例

完全な例は [examples](examples/) ディレクトリを参照してください。

```bash
cd examples
go run main.go
```

テストリクエスト：

```bash
# ユーザーを作成 (JSONC)
curl -X POST http://localhost:8080/api/v1/users \
  -H 'Content-Type: application/jsonc' \
  -d '{"name":"Alice","email":"alice@example.com","age":25}'

# ユーザーリストを取得
curl http://localhost:8080/api/v1/users

# 単一ユーザーを取得
curl http://localhost:8080/api/v1/users/1
```

## 依存関係

- [gin-gonic/gin](https://github.com/gin-gonic/gin) - Web フレームワーク
- [metamessage/metamessage](https://github.com/metamessage/metamessage) - MetaMessage プロトコル実装

## ライセンス

MIT

---

**[中文](README.md)** | [English](README.en.md) | 日本語
