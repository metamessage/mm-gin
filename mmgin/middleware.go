package mmgin

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	mm "github.com/metamessage/metamessage"
	"github.com/metamessage/mm-web-go/web"
)

// DecodeConfig configures request body decoding behavior.
type DecodeConfig struct {
	// AllowJSONC enables JSONC format request parsing.
	AllowJSONC bool
	// AllowMetaMessage enables MetaMessage binary format request parsing.
	AllowMetaMessage bool
	// DefaultFormat specifies the parsing format when Content-Type cannot be determined.
	DefaultFormat FormatType
	// MaxBodySize is the maximum request body size in bytes. 0 means unlimited.
	MaxBodySize int64
}

// FormatType represents the data format type for encoding/decoding.
type FormatType int

const (
	FormatAuto FormatType = iota
	FormatJSONC
	FormatMetaMessage
)

// EncodeConfig configures response encoding behavior.
type EncodeConfig struct {
	// DefaultFormat specifies the default response format.
	DefaultFormat FormatType
	// AutoNegotiate determines whether to auto-select format based on Accept header.
	AutoNegotiate bool
	// SuccessCode is the HTTP status code for successful responses.
	SuccessCode int
}

// DefaultDecodeConfig returns the default decode configuration.
func DefaultDecodeConfig() *DecodeConfig {
	return &DecodeConfig{
		AllowJSONC:       true,
		AllowMetaMessage: true,
		DefaultFormat:    FormatAuto,
		MaxBodySize:      10 << 20, // 10MB
	}
}

// DefaultEncodeConfig returns the default encode configuration.
func DefaultEncodeConfig() *EncodeConfig {
	return &EncodeConfig{
		DefaultFormat: FormatMetaMessage,
		AutoNegotiate: false,
		SuccessCode:   http.StatusOK,
	}
}

// mmError represents an error response in MetaMessage format.
type mmError struct {
	Error string `mm:"desc=錯誤信息"`
}

// MMResp wraps response data with an optional mm tag for MetaMessage encoding.
type MMResp struct {
	Data any
	Tag  string
}

// MetaMessageDecoder is a middleware that decodes request bodies.
// It supports both JSONC and MetaMessage binary formats, decoding into the target struct.
func MetaMessageDecoder(config *DecodeConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultDecodeConfig()
	}

	return func(c *gin.Context) {
		// Skip methods that do not carry a request body
		if c.Request.Method == http.MethodGet ||
			c.Request.Method == http.MethodHead ||
			c.Request.Method == http.MethodDelete ||
			c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		contentType := c.ContentType()
		format := detectFormat(contentType, config.DefaultFormat)

		var body []byte
		var err error

		if config.MaxBodySize > 0 {
			body, err = io.ReadAll(io.LimitReader(c.Request.Body, config.MaxBodySize+1))
			if int64(len(body)) > config.MaxBodySize {
				AbortWithMetaMessage(c, http.StatusRequestEntityTooLarge, mmError{
					Error: fmt.Sprintf("request body too large: %d > %d", int64(len(body)), config.MaxBodySize),
				})
				return
			}
		} else {
			body, err = io.ReadAll(c.Request.Body)
		}

		if err != nil {
			AbortWithMetaMessage(c, http.StatusBadRequest, mmError{
				Error: fmt.Sprintf("failed to read request body: %s", err.Error()),
			})
			return
		}

		// Restore Request.Body so subsequent middleware can re-read it
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		// Store raw body and format in context for later binding
		c.Set("mm_raw_body", body)
		c.Set("mm_format", format)

		c.Next()
	}
}

// Bind binds the request body to the target struct.
// Usage in handler: var req MyRequest; if err := mmgin.Bind(c, &req); err != nil { ... }
func Bind(c *gin.Context, obj any) error {
	body, exists := c.Get("mm_raw_body")
	if !exists {
		// Read request body directly if no prior decode middleware ran
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

	// Decode based on detected format
	switch format {
	case FormatMetaMessage:
		return mm.DecodeToValue(data, obj)
	case FormatJSONC:
		jsoncStr := string(data)
		return mm.JsoncToValue(jsoncStr, obj)
	default:
		// Auto-detect format from binary content
		if len(data) > 0 && isBinaryMetaMessage(data) {
			return mm.DecodeToValue(data, obj)
		}
		return mm.JsoncToValue(string(data), obj)
	}
}

// BindWithTag binds the request body to the target struct using the specified mm tag.
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

// MetaMessageEncoder is a middleware that encodes response data.
// It encodes the response set by handlers into MetaMessage binary format.
func MetaMessageEncoder(config *EncodeConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultEncodeConfig()
	}

	return func(c *gin.Context) {
		c.Next()

		// Skip encoding if the request was aborted or response was already written
		if c.IsAborted() || c.Writer.Written() {
			return
		}

		// Retrieve response data set by handler
		data, exists := c.Get("mm_response")
		if !exists {
			return
		}

		// Always encode to MetaMessage binary format
		mmResp := data.(MMResp)
		encoded, err := mm.EncodeFromValue(mmResp.Data, mmResp.Tag)
		if err != nil {
			AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{
				Error: fmt.Sprintf("failed to encode response: %s", err.Error()),
			})
			return
		}

		c.Data(config.SuccessCode, web.ContentTypeMetaMessage, encoded)
	}
}

// Respond sets response data for MetaMessage encoding (used by handlers).
func Respond(c *gin.Context, data any, tag string) {
	c.Set("mm_response", MMResp{Data: data, Tag: tag})
}

// RespondWithStatus sets response data and HTTP status code.
func RespondWithStatus(c *gin.Context, code int, data any, tag string) {
	c.Set("mm_response", MMResp{Data: data, Tag: tag})
	c.Status(code)
}

// AbortWithMetaMessage sends a MetaMessage-format error response and aborts the request.
// All errors use MetaMessage format to ensure consistent response format.
// Falls back to a minimal error struct even if encoding itself fails.
func AbortWithMetaMessage(c *gin.Context, code int, obj any) {
	encoded, err := mm.EncodeFromValue(obj, "")
	if err != nil {
		// Retry with a minimal fallback struct on encoding failure
		fallback := struct {
			Error string `mm:"desc=錯誤信息"`
		}{Error: fmt.Sprintf("internal server error: %s", err.Error())}
		encoded, _ = mm.EncodeFromValue(fallback, "")
	}
	c.Status(code)
	c.Header("Content-Type", web.ContentTypeMetaMessage)
	_, _ = c.Writer.Write(encoded)
	c.Abort()
}

// OptionsHandler returns a handler for OPTIONS requests.
// Used for schema discovery: clients send OPTIONS to receive the request struct schema.
// Returns a MetaMessage-encoded struct instance with full type, constraint, and description info.
func OptionsHandler(obj any) gin.HandlerFunc {
	return func(c *gin.Context) {
		encoded, err := mm.EncodeFromValue(obj, "")
		if err != nil {
			AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{
				Error: fmt.Sprintf("failed to encode schema: %s", err.Error()),
			})
			return
		}
		c.Data(http.StatusOK, web.ContentTypeMetaMessage, encoded)
	}
}

// RouteGroup wraps gin.RouterGroup for extended routing capabilities.
type RouteGroup struct {
	*gin.RouterGroup
}

// Group creates a RouteGroup from a gin.RouterGroup.
func Group(rg *gin.RouterGroup) *RouteGroup {
	return &RouteGroup{RouterGroup: rg}
}

var defaultGroup *gin.RouterGroup

// Init initializes MetaMessage middleware and a route group.
// It registers MetaMessageDecoder and MetaMessageEncoder internally,
// and sets the returned group as the default for all route registration functions
// (GET, POST, PUT, DELETE, etc.). After Init, use mmgin.GET, mmgin.POST directly.
func Init(r *gin.Engine, relativePath string) *gin.RouterGroup {
	r.Use(MetaMessageDecoder(nil))
	r.Use(MetaMessageEncoder(nil))
	rg := r.Group(relativePath)
	defaultGroup = rg
	return rg
}

// Handler defines a generic handler type with automatic request binding.
// T is the request struct type; the handler receives a *T pointer.
// Returns data, tag string, and error; the framework handles Respond/AbortWithMetaMessage.
type Handler[T any] func(c *gin.Context, req *T) (data any, tag string, err error)

// GET registers a GET route with query parameter binding via MetaMessage.
func GET[T any](relativePath string, handler Handler[T]) {
	if defaultGroup == nil {
		panic("mmgin: Init() must be called before GET()")
	}
	defaultGroup.Handle("GET", relativePath, func(c *gin.Context) {
		var req T
		if err := BindQuery(c, &req); err != nil {
			AbortWithMetaMessage(c, http.StatusBadRequest, mmError{
				Error: "bind failed: " + err.Error(),
			})
			return
		}
		data, tag, err := handler(c, &req)
		if err != nil {
			AbortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		Respond(c, data, tag)
	})
	defaultGroup.OPTIONS(relativePath, func(c *gin.Context) {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{
				Error: fmt.Sprintf("failed to encode schema: %s", err.Error()),
			})
			return
		}
		c.Header("Allow", "GET, OPTIONS")
		c.Data(http.StatusOK, web.ContentTypeMetaMessage, encoded)
	})
}

// HEAD registers a HEAD route.
func HEAD(relativePath string, handlers ...gin.HandlerFunc) {
	if defaultGroup == nil {
		panic("mmgin: Init() must be called before HEAD()")
	}
	defaultGroup.HEAD(relativePath, handlers...)
}

// DELETE registers a DELETE route with query parameter binding via MetaMessage.
func DELETE[T any](relativePath string, handler Handler[T]) {
	if defaultGroup == nil {
		panic("mmgin: Init() must be called before DELETE()")
	}
	defaultGroup.Handle("DELETE", relativePath, func(c *gin.Context) {
		var req T
		if err := BindQuery(c, &req); err != nil {
			AbortWithMetaMessage(c, http.StatusBadRequest, mmError{
				Error: "bind failed: " + err.Error(),
			})
			return
		}
		data, tag, err := handler(c, &req)
		if err != nil {
			AbortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		Respond(c, data, tag)
	})
	defaultGroup.OPTIONS(relativePath, func(c *gin.Context) {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{
				Error: fmt.Sprintf("failed to encode schema: %s", err.Error()),
			})
			return
		}
		c.Header("Allow", "DELETE, OPTIONS")
		c.Data(http.StatusOK, web.ContentTypeMetaMessage, encoded)
	})
}

// OPTIONS registers an OPTIONS route.
func OPTIONS(relativePath string, handlers ...gin.HandlerFunc) {
	if defaultGroup == nil {
		panic("mmgin: Init() must be called before OPTIONS()")
	}
	defaultGroup.OPTIONS(relativePath, handlers...)
}

// Any registers a route for all HTTP methods.
func Any(relativePath string, handlers ...gin.HandlerFunc) {
	if defaultGroup == nil {
		panic("mmgin: Init() must be called before Any()")
	}
	defaultGroup.Any(relativePath, handlers...)
}

// POST registers a POST route with auto-binding and automatically registers
// an OPTIONS route on the same path for schema discovery.
// Must be called after Init().
func POST[T any](relativePath string, handler Handler[T]) {
	if defaultGroup == nil {
		panic("mmgin: Init() must be called before POST()")
	}
	defaultGroup.Handle("POST", relativePath, func(c *gin.Context) {
		var req T
		if err := MustBindAndValidate(c, &req); err != nil {
			AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{
				Error: fmt.Sprintf("binding failed: %s", err.Error()),
			})
			return
		}
		data, tag, err := handler(c, &req)
		if err != nil {
			AbortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		Respond(c, data, tag)
	})
	defaultGroup.OPTIONS(relativePath, func(c *gin.Context) {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{
				Error: fmt.Sprintf("failed to encode schema: %s", err.Error()),
			})
			return
		}
		c.Header("Allow", "POST, OPTIONS")
		c.Data(http.StatusOK, web.ContentTypeMetaMessage, encoded)
	})
}

// PUT registers a PUT route with auto-binding and automatically registers
// an OPTIONS route on the same path for schema discovery.
// Must be called after Init().
func PUT[T any](relativePath string, handler Handler[T]) {
	if defaultGroup == nil {
		panic("mmgin: Init() must be called before PUT()")
	}
	defaultGroup.Handle("PUT", relativePath, func(c *gin.Context) {
		var req T
		if err := MustBindAndValidate(c, &req); err != nil {
			AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{
				Error: fmt.Sprintf("binding failed: %s", err.Error()),
			})
			return
		}
		data, tag, err := handler(c, &req)
		if err != nil {
			AbortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		Respond(c, data, tag)
	})
	defaultGroup.OPTIONS(relativePath, func(c *gin.Context) {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{
				Error: fmt.Sprintf("failed to encode schema: %s", err.Error()),
			})
			return
		}
		c.Header("Allow", "PUT, OPTIONS")
		c.Data(http.StatusOK, web.ContentTypeMetaMessage, encoded)
	})
}

// PATCH registers a PATCH route with auto-binding and automatically registers
// an OPTIONS route on the same path for schema discovery.
// Must be called after Init().
func PATCH[T any](relativePath string, handler Handler[T]) {
	if defaultGroup == nil {
		panic("mmgin: Init() must be called before PATCH()")
	}
	defaultGroup.Handle("PATCH", relativePath, func(c *gin.Context) {
		var req T
		if err := MustBindAndValidate(c, &req); err != nil {
			AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{
				Error: fmt.Sprintf("binding failed: %s", err.Error()),
			})
			return
		}
		data, tag, err := handler(c, &req)
		if err != nil {
			AbortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		Respond(c, data, tag)
	})
	defaultGroup.OPTIONS(relativePath, func(c *gin.Context) {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{
				Error: fmt.Sprintf("failed to encode schema: %s", err.Error()),
			})
			return
		}
		c.Header("Allow", "PATCH, OPTIONS")
		c.Data(http.StatusOK, web.ContentTypeMetaMessage, encoded)
	})
}

// detectFormat detects the data format from the Content-Type header.
func detectFormat(contentType string, defaultFormat FormatType) FormatType {
	switch contentType {
	case web.ContentTypeMetaMessage, "application/octet-stream":
		return FormatMetaMessage
	case web.ContentTypeJSONC, "application/json", "text/plain":
		return FormatJSONC
	default:
		if defaultFormat != FormatAuto {
			return defaultFormat
		}
		return FormatJSONC // Default to JSONC
	}
}

// isBinaryMetaMessage detects whether the data is in MetaMessage binary format.
func isBinaryMetaMessage(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	// Valid MetaMessage binary does not start with JSON delimiters ({, [, ")
	firstChar := data[0]
	return firstChar != '{' && firstChar != '[' && firstChar != '"'
}
