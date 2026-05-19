# gin-mm

Gin 框架的 MetaMessage 插件，提供編解碼、數據綁定等功能。

## 特性

- **請求解碼**：自動檢測並解碼 JSONC 和 MetaMessage 二進制格式的請求體
- **響應編碼**：自動將響應數據編碼為 JSONC 或 MetaMessage 格式
- **數據綁定**：支持請求體、查詢參數、URI 參數、請求頭的綁定
- **內容協商**：根據 Accept 頭自動選擇響應格式
- **自定義驗證**：支持結構體自定義驗證邏輯

## 安裝

```bash
go get github.com/metamessage/gin-mm
```

## 快速開始

```go
package main

import (
    "github.com/gin-gonic/gin"
    ginmm "github.com/metamessage/gin-mm"
)

type User struct {
    Name  string `mm:"type=str;desc=用戶名稱" json:"name"`
    Email string `mm:"type=email;desc=電子郵箱" json:"email"`
    Age   int    `mm:"type=uint8;desc=年齡" json:"age"`
}

func main() {
    r := gin.Default()

    // 使用全局中間件
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

## API 文檔

### 中間件

#### MetaMessageDecoder

請求體解碼中間件，支持 JSONC 和 MetaMessage 二進制格式。

```go
// 使用默認配置
r.Use(ginmm.MetaMessageDecoder(nil))

// 自定義配置
config := &ginmm.DecodeConfig{
    AllowJSONC:       true,
    AllowMetaMessage: true,
    DefaultFormat:    ginmm.FormatAuto,
    MaxBodySize:      10 << 20, // 10MB
}
r.Use(ginmm.MetaMessageDecoder(config))
```

#### MetaMessageEncoder

響應編碼中間件，自動將響應數據編碼為指定格式。

```go
// 使用默認配置
r.Use(ginmm.MetaMessageEncoder(nil))

// 自定義配置
config := &ginmm.EncodeConfig{
    DefaultFormat: ginmm.FormatMetaMessage,
    AutoNegotiate: true,
    SuccessCode:   200,
}
r.Use(ginmm.MetaMessageEncoder(config))
```

### 數據綁定

#### Bind

將請求體綁定到結構體。

```go
var user User
if err := ginmm.Bind(c, &user); err != nil {
    // 處理錯誤
}
```

#### MustBind

綁定數據，失敗時自動返回 400 錯誤響應。

```go
var user User
if err := ginmm.MustBind(c, &user); err != nil {
    return // 錯誤響應已自動處理
}
```

#### BindQuery

綁定查詢參數到結構體。

```go
var filter Filter
if err := ginmm.BindQuery(c, &filter); err != nil {
    // 處理錯誤
}
```

#### BindUri

綁定 URI 參數到結構體。

```go
var params Params
ginmm.BindUri(c, &params)
```

#### AutoBind

自動綁定所有數據源（URI 參數 > 查詢參數 > 請求體）。

```go
var req Request
if err := ginmm.AutoBind(c, &req); err != nil {
    // 處理錯誤
}
```

### 數據驗證

#### 自定義驗證器

實現 `Validator` 接口：

```go
type CreateUserRequest struct {
    Name string `mm:"type=str" json:"name"`
    Age  int    `mm:"type=uint8" json:"age"`
}

func (r *CreateUserRequest) Validate() error {
    if r.Age < 18 {
        return errors.New("用戶必須年滿18歲")
    }
    return nil
}
```

#### BindAndValidate

綁定並驗證數據：

```go
var req CreateUserRequest
if err := ginmm.BindAndValidate(c, &req); err != nil {
    // 處理錯誤
}
```

#### MustBindAndValidate

綁定並驗證，失敗時自動返回錯誤響應：

```go
var req CreateUserRequest
if err := ginmm.MustBindAndValidate(c, &req); err != nil {
    return // 錯誤響應已自動處理
}
```

### 響應函數

#### Respond

設置響應數據（由編碼中間件處理）。

```go
ginmm.Respond(c, data)
```

#### RespondWithStatus

設置響應數據和狀態碼。

```go
ginmm.RespondWithStatus(c, http.StatusCreated, data)
```

#### JSONC

直接返回 JSONC 格式響應。

```go
ginmm.JSONC(c, 200, data)
```

#### MetaMessage

直接返回 MetaMessage 二進制格式響應。

```go
ginmm.MetaMessage(c, 200, data)
```

## 配置選項

### DecodeConfig

| 字段 | 類型 | 默認值 | 說明 |
|------|------|--------|------|
| AllowJSONC | bool | true | 是否允許 JSONC 格式請求 |
| AllowMetaMessage | bool | true | 是否允許 MetaMessage 二進制格式請求 |
| DefaultFormat | FormatType | FormatAuto | 默認解析格式 |
| MaxBodySize | int64 | 10MB | 最大請求體大小 |

### EncodeConfig

| 字段 | 類型 | 默認值 | 說明 |
|------|------|--------|------|
| DefaultFormat | FormatType | FormatMetaMessage | 默認響應格式 |
| AutoNegotiate | bool | true | 是否根據 Accept 頭自動選擇格式 |
| SuccessCode | int | 200 | 成功響應狀態碼 |

### FormatType

```go
FormatAuto          // 自動檢測
FormatJSONC         // JSONC 格式
FormatMetaMessage   // MetaMessage 二進制格式
```

## Content-Type

- `application/x-metamessage` - MetaMessage 二進制格式
- `application/jsonc` - JSONC 格式

## 示例

查看 [examples](examples/) 目錄獲取完整示例。

```bash
cd examples
go run main.go
```

測試請求：

```bash
# 創建用戶（JSONC）
curl -X POST http://localhost:8080/api/v1/users \
  -H 'Content-Type: application/jsonc' \
  -d '{"name":"Alice","email":"alice@example.com","age":25}'

# 獲取用戶列表
curl http://localhost:8080/api/v1/users

# 獲取單個用戶
curl http://localhost:8080/api/v1/users/1
```

## 依賴

- [gin-gonic/gin](https://github.com/gin-gonic/gin) - Web 框架
- [metamessage/metamessage](https://github.com/metamessage/metamessage) - MetaMessage 協議實現

## License

MIT
