package mmfiber

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/gofiber/fiber/v2"
	mm "github.com/metamessage/metamessage"
)

const (
	ContentTypeMetaMessage = "application/x-metamessage"
	ContentTypeJSONC       = "application/jsonc"
)

var defaultGroup fiber.Router

type mmError struct {
	Error string `mm:"desc=Error info"`
}

func decoderMiddleware(c *fiber.Ctx) error {
	if c.Method() == http.MethodGet || c.Method() == http.MethodHead ||
		c.Method() == http.MethodDelete || c.Method() == http.MethodOptions {
		return c.Next()
	}
	body := c.Body()
	if len(body) > 0 {
		c.Locals("mm_raw_body", body)
	}
	return c.Next()
}

func Init(app *fiber.App, relativePath string) fiber.Router {
	g := app.Group(relativePath, func(c *fiber.Ctx) error {
		return c.Next()
	})
	defaultGroup = g
	return g
}

func GET(relativePath string, handlers ...fiber.Handler) {
	defaultGroup.Get(relativePath, handlers...)
}

func HEAD(relativePath string, handlers ...fiber.Handler) {
	defaultGroup.Head(relativePath, handlers...)
}

func DELETE(relativePath string, handlers ...fiber.Handler) {
	defaultGroup.Delete(relativePath, handlers...)
}

func OPTIONS(relativePath string, handlers ...fiber.Handler) {
	defaultGroup.Options(relativePath, handlers...)
}

func Any(relativePath string, handlers ...fiber.Handler) {
	defaultGroup.All(relativePath, handlers...)
}

type Handler[T any] func(c *fiber.Ctx, req *T) (data any, err error)

func POST[T any](relativePath string, handler Handler[T]) {
	defaultGroup.Post(relativePath, func(c *fiber.Ctx) error {
		var req T
		if err := bind(c, &req); err != nil {
			return abortWithMetaMessage(c, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
		}
		data, err := handler(c, &req)
		if err != nil {
			return abortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
		}
		respond(c, data, "")
		return nil
	})
	defaultGroup.Options(relativePath, func(c *fiber.Ctx) error {
		var sample T
		initSafeDefaults(&sample)
		encoded, err := mm.EncodeFromValue(sample, "")
		if err != nil {
			return abortWithMetaMessage(c, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
		}
		c.Set("Allow", "POST, OPTIONS")
		c.Set("Content-Type", ContentTypeMetaMessage)
		return c.Status(http.StatusOK).Send(encoded)
	})
}

func PUT[T any](relativePath string, handler Handler[T]) {
	defaultGroup.Put(relativePath, func(c *fiber.Ctx) error {
		var req T
		if err := bind(c, &req); err != nil {
			return abortWithMetaMessage(c, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
		}
		data, err := handler(c, &req)
		if err != nil {
			return abortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
		}
		respond(c, data, "")
		return nil
	})
	defaultGroup.Options(relativePath, func(c *fiber.Ctx) error {
		var sample T
		initSafeDefaults(&sample)
		encoded, err := mm.EncodeFromValue(sample, "")
		if err != nil {
			return abortWithMetaMessage(c, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
		}
		c.Set("Allow", "PUT, OPTIONS")
		c.Set("Content-Type", ContentTypeMetaMessage)
		return c.Status(http.StatusOK).Send(encoded)
	})
}

func PATCH[T any](relativePath string, handler Handler[T]) {
	defaultGroup.Patch(relativePath, func(c *fiber.Ctx) error {
		var req T
		if err := bind(c, &req); err != nil {
			return abortWithMetaMessage(c, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
		}
		data, err := handler(c, &req)
		if err != nil {
			return abortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
		}
		respond(c, data, "")
		return nil
	})
	defaultGroup.Options(relativePath, func(c *fiber.Ctx) error {
		var sample T
		initSafeDefaults(&sample)
		encoded, err := mm.EncodeFromValue(sample, "")
		if err != nil {
			return abortWithMetaMessage(c, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
		}
		c.Set("Allow", "PATCH, OPTIONS")
		c.Set("Content-Type", ContentTypeMetaMessage)
		return c.Status(http.StatusOK).Send(encoded)
	})
}

func bind(c *fiber.Ctx, obj any) error {
	raw, ok := c.Locals("mm_raw_body").([]byte)
	if !ok {
		body := c.Body()
		if len(body) == 0 {
			return fmt.Errorf("empty body")
		}
		raw = body
	}
	if isBinaryMetaMessage(raw) {
		return mm.DecodeToValue(raw, obj)
	}
	return mm.JsoncToValue(string(raw), obj)
}

func isBinaryMetaMessage(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	c := data[0]
	return c != '{' && c != '[' && c != '"'
}

func respond(c *fiber.Ctx, data any, tag string) {
	c.Locals("mm_response", data)
	c.Locals("mm_tag", tag)
	err := respondFromLocals(c)
	if err != nil {
		c.Locals("mm_error", err)
	}
}

func respondWithStatus(c *fiber.Ctx, code int, data any, tag string) {
	c.Locals("mm_response", data)
	c.Locals("mm_status", code)
	c.Locals("mm_tag", tag)
}

func abortWithMetaMessage(c *fiber.Ctx, code int, obj any) error {
	encoded, err := mm.EncodeFromValue(obj, "")
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("encode error")
	}
	c.Set("Content-Type", ContentTypeMetaMessage)
	return c.Status(code).Send(encoded)
}

func respondFromLocals(c *fiber.Ctx) error {
	mmData := c.Locals("mm_response")
	if mmData == nil {
		return nil
	}
	tag := ""
	if t, ok := c.Locals("mm_tag").(string); ok {
		tag = t
	}
	encoded, err := mm.EncodeFromValue(mmData, tag)
	if err != nil {
		return err
	}
	status := http.StatusOK
	if s, ok := c.Locals("mm_status").(int); ok {
		status = s
	}
	c.Set("Content-Type", ContentTypeMetaMessage)
	return c.Status(status).Send(encoded)
}

func initSafeDefaults(obj any) {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() {
			continue
		}
		tag := t.Field(i).Tag.Get("mm")
		switch field.Kind() {
		case reflect.String:
			if field.String() == "" {
				field.SetString("x")
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if field.Int() == 0 {
				field.SetInt(1)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if field.Uint() == 0 {
				field.SetUint(1)
			}
		case reflect.Float32, reflect.Float64:
			if field.Float() == 0 {
				_ = tag
				field.SetFloat(1)
			}
		}
	}
}