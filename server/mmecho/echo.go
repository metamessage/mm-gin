package mmecho

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/metamessage/web"

	echo "github.com/labstack/echo/v4"
	mm "github.com/metamessage/metamessage"
)

var defaultGroup *echo.Group

type mmError struct {
	Error string `mm:"desc=Error info"`
}

func decoderMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Request().Method == http.MethodGet || c.Request().Method == http.MethodHead ||
			c.Request().Method == http.MethodDelete || c.Request().Method == http.MethodOptions {
			return next(c)
		}
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return AbortWithMetaMessage(c, http.StatusBadRequest, mmError{Error: "read body failed"})
		}
		c.Request().Body = io.NopCloser(bytes.NewReader(body))
		c.Set("mm_raw_body", body)
		return next(c)
	}
}

func encoderMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := next(c)
		if mmData := c.Get("mm_response"); mmData != nil {
			tag := ""
			if t, ok := c.Get("mm_tag").(string); ok {
				tag = t
			}
			encoded, mmErr := mm.EncodeFromValue(mmData, tag)
			if mmErr != nil {
				return mmErr
			}
			status := http.StatusOK
			if s, ok := c.Get("mm_status").(int); ok {
				status = s
			}
			return c.Blob(status, web.ContentTypeMetaMessage, encoded)
		}
		return err
	}
}

func Init(e *echo.Echo, relativePath string) *echo.Group {
	e.Use(decoderMiddleware)
	e.Use(encoderMiddleware)
	g := e.Group(relativePath)
	defaultGroup = g
	return g
}

func GET(relativePath string, handlers ...echo.HandlerFunc) {
	if len(handlers) == 0 {
		return
	}
	h := handlers[0]
	defaultGroup.GET(relativePath, h)
}

func HEAD(relativePath string, handlers ...echo.HandlerFunc) {
	if len(handlers) == 0 {
		return
	}
	h := handlers[0]
	defaultGroup.HEAD(relativePath, h)
}

func DELETE(relativePath string, handlers ...echo.HandlerFunc) {
	if len(handlers) == 0 {
		return
	}
	h := handlers[0]
	defaultGroup.DELETE(relativePath, h)
}

func OPTIONS(relativePath string, handlers ...echo.HandlerFunc) {
	if len(handlers) == 0 {
		return
	}
	h := handlers[0]
	defaultGroup.OPTIONS(relativePath, h)
}

func Any(relativePath string, handlers ...echo.HandlerFunc) {
	if len(handlers) == 0 {
		return
	}
	h := handlers[0]
	defaultGroup.Any(relativePath, h)
}

type Handler[T any] func(c echo.Context, req *T) (data any, err error)

func POST[T any](relativePath string, handler Handler[T]) {
	defaultGroup.POST(relativePath, func(c echo.Context) error {
		var req T
		if err := bind(c, &req); err != nil {
			return AbortWithMetaMessage(c, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
		}
		data, err := handler(c, &req)
		if err != nil {
			return AbortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
		}
		Respond(c, data, "")
		return nil
	})
	defaultGroup.OPTIONS(relativePath, func(c echo.Context) error {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			return AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
		}
		c.Response().Header().Set("Allow", "POST, OPTIONS")
		return c.Blob(http.StatusOK, web.ContentTypeMetaMessage, encoded)
	})
}

func PUT[T any](relativePath string, handler Handler[T]) {
	defaultGroup.PUT(relativePath, func(c echo.Context) error {
		var req T
		if err := bind(c, &req); err != nil {
			return AbortWithMetaMessage(c, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
		}
		data, err := handler(c, &req)
		if err != nil {
			return AbortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
		}
		Respond(c, data, "")
		return nil
	})
	defaultGroup.OPTIONS(relativePath, func(c echo.Context) error {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			return AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
		}
		c.Response().Header().Set("Allow", "PUT, OPTIONS")
		return c.Blob(http.StatusOK, web.ContentTypeMetaMessage, encoded)
	})
}

func PATCH[T any](relativePath string, handler Handler[T]) {
	defaultGroup.PATCH(relativePath, func(c echo.Context) error {
		var req T
		if err := bind(c, &req); err != nil {
			return AbortWithMetaMessage(c, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
		}
		data, err := handler(c, &req)
		if err != nil {
			return AbortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
		}
		Respond(c, data, "")
		return nil
	})
	defaultGroup.OPTIONS(relativePath, func(c echo.Context) error {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			return AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
		}
		c.Response().Header().Set("Allow", "PATCH, OPTIONS")
		return c.Blob(http.StatusOK, web.ContentTypeMetaMessage, encoded)
	})
}

func bind(c echo.Context, obj any) error {
	body, exists := c.Get("mm_raw_body").([]byte)
	if !exists {
		var err error
		body, err = io.ReadAll(c.Request().Body)
		if err != nil {
			return fmt.Errorf("read body: %w", err)
		}
		c.Request().Body = io.NopCloser(bytes.NewReader(body))
	}
	if isBinaryMetaMessage(body) {
		return mm.DecodeToValue(body, obj)
	}
	return mm.JsoncToValue(string(body), obj)
}

func isBinaryMetaMessage(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	c := data[0]
	return c != '{' && c != '[' && c != '"'
}

func Respond(c echo.Context, data any, tag string) {
	c.Set("mm_response", data)
	c.Set("mm_tag", tag)
}

func RespondWithStatus(c echo.Context, code int, data any, tag string) {
	c.Set("mm_response", data)
	c.Set("mm_status", code)
	c.Set("mm_tag", tag)
}

func AbortWithMetaMessage(c echo.Context, code int, obj any) error {
	encoded, err := mm.EncodeFromValue(obj, "")
	if err != nil {
		return c.String(http.StatusInternalServerError, "encode error")
	}
	return c.Blob(code, web.ContentTypeMetaMessage, encoded)
}
