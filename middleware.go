package ginmm

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	mm "github.com/metamessage/metamessage"
)

// ContentTypeMetaMessage MetaMessage 二進制格式 Content-Type
const ContentTypeMetaMessage = "application/x-metamessage"

// ContentTypeJSONC JSONC 格式 Content-Type
const ContentTypeJSONC = "application/jsonc"

// DecodeConfig 請求解碼配置
type DecodeConfig struct {
	// 是否允許 JSONC 格式請求
	AllowJSONC bool
	// 是否允許 MetaMessage 二進制格式請求
	AllowMetaMessage bool
	// 默認解析格式（當無法從 Content-Type 判斷時）
	DefaultFormat FormatType
	// 最大請求體大小（字節），0 表示不限制
	MaxBodySize int64
}

// FormatType 數據格式類型
type FormatType int

const (
	FormatAuto FormatType = iota
	FormatJSONC
	FormatMetaMessage
)

// EncodeConfig 響應編碼配置
type EncodeConfig struct {
	// 默認響應格式
	DefaultFormat FormatType
	// 是否根據請求的 Accept 頭自動選擇格式
	AutoNegotiate bool
	// 響應狀態碼
	SuccessCode int
}

// DefaultDecodeConfig 默認解碼配置
func DefaultDecodeConfig() *DecodeConfig {
	return &DecodeConfig{
		AllowJSONC:       true,
		AllowMetaMessage: true,
		DefaultFormat:    FormatAuto,
		MaxBodySize:      10 << 20, // 10MB
	}
}

// DefaultEncodeConfig 默認編碼配置
func DefaultEncodeConfig() *EncodeConfig {
	return &EncodeConfig{
		DefaultFormat: FormatMetaMessage,
		AutoNegotiate: false,
		SuccessCode:   http.StatusOK,
	}
}

// mmError 用於 MetaMessage 格式的錯誤響應
type mmError struct {
	Error string `mm:"desc=錯誤信息"`
}

type MMResp struct {
	Data any
	Tag  string
}

// MetaMessageDecoder 請求體解碼中間件
// 將請求體（JSONC 或 MetaMessage 二進制）解碼到指定的結構體
func MetaMessageDecoder(config *DecodeConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultDecodeConfig()
	}

	return func(c *gin.Context) {
		// 只處理有請求體的方法
		if c.Request.Method == http.MethodGet ||
			c.Request.Method == http.MethodHead ||
			c.Request.Method == http.MethodDelete ||
			c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		// 檢查 Content-Type
		contentType := c.ContentType()
		format := detectFormat(contentType, config.DefaultFormat)

		// 讀取請求體
		var body []byte
		var err error

		if config.MaxBodySize > 0 {
			body, err = io.ReadAll(io.LimitReader(c.Request.Body, config.MaxBodySize+1))
			if int64(len(body)) > config.MaxBodySize {
				AbortWithMetaMessage(c, http.StatusRequestEntityTooLarge, mmError{
					Error: "request body too large",
				})
				return
			}
		} else {
			body, err = io.ReadAll(c.Request.Body)
		}

		if err != nil {
			AbortWithMetaMessage(c, http.StatusBadRequest, mmError{
				Error: "failed to read request body",
			})
			return
		}

		// 重新設置 Request.Body 以便後續中間件可以讀取
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		// 存儲原始請求體和格式信息到上下文
		c.Set("mm_raw_body", body)
		c.Set("mm_format", format)

		c.Next()
	}
}

// Bind 將請求體綁定到目標結構體
// 在 handler 中使用：var req MyRequest; if err := ginmm.Bind(c, &req); err != nil { ... }
func Bind(c *gin.Context, obj any) error {
	body, exists := c.Get("mm_raw_body")
	if !exists {
		// 如果沒有經過解碼中間件，直接讀取請求體
		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return err
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(data))
		body = data
	}

	formatVal, _ := c.Get("mm_format")
	format, _ := formatVal.(FormatType)

	data := body.([]byte)

	// 根據格式選擇解碼方式
	switch format {
	case FormatMetaMessage:
		return mm.DecodeToValue(data, obj)
	case FormatJSONC:
		jsoncStr := string(data)
		return mm.JsoncToValue(jsoncStr, obj)
	default:
		// 自動檢測格式
		if len(data) > 0 && isBinaryMetaMessage(data) {
			return mm.DecodeToValue(data, obj)
		}
		return mm.JsoncToValue(string(data), obj)
	}
}

// BindWithTag 使用指定的 tag 將請求體綁定到目標結構體
func BindWithTag(c *gin.Context, obj any, tag string) error {
	body, exists := c.Get("mm_raw_body")
	if !exists {
		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return err
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(data))
		body = data
	}

	formatVal, _ := c.Get("mm_format")
	format, _ := formatVal.(FormatType)

	data := body.([]byte)

	switch format {
	case FormatMetaMessage:
		return mm.DecodeToValue(data, obj)
	case FormatJSONC:
		jsoncStr := string(data)
		return mm.JsoncToValue(jsoncStr, obj)
	default:
		if len(data) > 0 && isBinaryMetaMessage(data) {
			return mm.DecodeToValue(data, obj)
		}
		return mm.JsoncToValue(string(data), obj)
	}
}

// MetaMessageEncoder 響應編碼中間件
// 將 handler 設置的響應數據編碼為 MetaMessage 二進制格式
func MetaMessageEncoder(config *EncodeConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultEncodeConfig()
	}

	return func(c *gin.Context) {
		c.Next()

		// 如果已經寫入響應或發生錯誤，不進行編碼
		if c.IsAborted() || c.Writer.Written() {
			return
		}

		// 獲取響應數據
		data, exists := c.Get("mm_response")
		if !exists {
			return
		}

		// 始終編碼為 MetaMessage 二進制格式
		mmResp := data.(MMResp)
		encoded, err := mm.EncodeFromValue(mmResp.Data, mmResp.Tag)
		if err != nil {
			AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{
				Error: fmt.Sprintf("failed to encode response: %s", err.Error()),
			})
			return
		}

		c.Data(config.SuccessCode, ContentTypeMetaMessage, encoded)
	}
}

// Respond 設置響應數據（供 handler 使用）
func Respond(c *gin.Context, data any, tag string) {
	c.Set("mm_response", MMResp{Data: data, Tag: tag})
}

// RespondWithStatus 設置響應數據和狀態碼
func RespondWithStatus(c *gin.Context, code int, data any, tag string) {
	c.Set("mm_response", MMResp{Data: data, Tag: tag})
	c.Status(code)
}

// AbortWithMetaMessage 返回 MetaMessage 格式的錯誤響應並中止請求
// 所有錯誤都使用 MetaMessage 格式，保證響應格式一致
// 即使編碼出錯也返回 MetaMessage 格式
func AbortWithMetaMessage(c *gin.Context, code int, obj any) {
	encoded, err := mm.EncodeFromValue(obj, "")
	if err != nil {
		// 編碼出錯時使用最簡結構重試
		fallback := struct {
			Error string `mm:"desc=錯誤信息"`
		}{Error: "internal server error"}
		encoded, _ = mm.EncodeFromValue(fallback, "")
	}
	c.Status(code)
	c.Header("Content-Type", ContentTypeMetaMessage)
	_, _ = c.Writer.Write(encoded)
	c.Abort()
}

// OptionsHandler 返回 OPTIONS 請求的處理函數
// 用於 Schema 發現：客戶端發送 OPTIONS 請求即可獲取請求體的結構信息
// 返回值是一個 MetaMessage 編碼的結構體實例，包含完整的類型、約束和描述
func OptionsHandler(obj any) gin.HandlerFunc {
	return func(c *gin.Context) {
		encoded, err := mm.EncodeFromValue(obj, "")
		if err != nil {
			AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{
				Error: "failed to encode schema",
			})
			return
		}
		c.Data(http.StatusOK, ContentTypeMetaMessage, encoded)
	}
}

// detectFormat 根據 Content-Type 檢測數據格式
func detectFormat(contentType string, defaultFormat FormatType) FormatType {
	switch contentType {
	case ContentTypeMetaMessage, "application/octet-stream":
		return FormatMetaMessage
	case ContentTypeJSONC, "application/json", "text/plain":
		return FormatJSONC
	default:
		if defaultFormat != FormatAuto {
			return defaultFormat
		}
		return FormatJSONC // 默認嘗試 JSONC
	}
}

// isBinaryMetaMessage 檢測數據是否為 MetaMessage 二進制格式
func isBinaryMetaMessage(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	// 檢查是否為有效的 MetaMessage 格式（非 JSON）
	// JSON 通常以 { 或 [ 開頭
	firstChar := data[0]
	return firstChar != '{' && firstChar != '[' && firstChar != '"'
}
