package mmfiber

import (
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/metamessage/mm-web-go/web"

	fiber "github.com/gofiber/fiber/v2"
	mm "github.com/metamessage/metamessage"
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

func bindQuery(c *fiber.Ctx, obj any) error {
	dataHex := c.Query("data")
	if dataHex == "" {
		return nil
	}
	binData, err := hex.DecodeString(dataHex)
	if err != nil {
		return fmt.Errorf("invalid hex string")
	}
	return mm.DecodeToValue(binData, obj)
}

func GET[T any](relativePath string, handler Handler[T]) {
	defaultGroup.Get(relativePath, func(c *fiber.Ctx) error {
		var req T
		if err := bindQuery(c, &req); err != nil {
			return AbortWithMetaMessage(c, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
		}
		data, tag, err := handler(c, &req)
		if err != nil {
			return AbortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
		}
		Respond(c, data, tag)
		return nil
	})
	defaultGroup.Options(relativePath, func(c *fiber.Ctx) error {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			return AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
		}
		c.Set("Allow", "GET, OPTIONS")
		c.Set("Content-Type", web.ContentTypeMetaMessage)
		return c.Status(http.StatusOK).Send(encoded)
	})
}

func DELETE[T any](relativePath string, handler Handler[T]) {
	defaultGroup.Delete(relativePath, func(c *fiber.Ctx) error {
		var req T
		if err := bindQuery(c, &req); err != nil {
			return AbortWithMetaMessage(c, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
		}
		data, tag, err := handler(c, &req)
		if err != nil {
			return AbortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
		}
		Respond(c, data, tag)
		return nil
	})
	defaultGroup.Options(relativePath, func(c *fiber.Ctx) error {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			return AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
		}
		c.Set("Allow", "DELETE, OPTIONS")
		c.Set("Content-Type", web.ContentTypeMetaMessage)
		return c.Status(http.StatusOK).Send(encoded)
	})
}

func HEAD(relativePath string, handlers ...fiber.Handler) {
	defaultGroup.Head(relativePath, handlers...)
}

func OPTIONS(relativePath string, handlers ...fiber.Handler) {
	defaultGroup.Options(relativePath, handlers...)
}

func Any(relativePath string, handlers ...fiber.Handler) {
	defaultGroup.All(relativePath, handlers...)
}

type Handler[T any] func(c *fiber.Ctx, req *T) (data any, tag string, err error)

func POST[T any](relativePath string, handler Handler[T]) {
	defaultGroup.Post(relativePath, func(c *fiber.Ctx) error {
		var req T
		if err := bind(c, &req); err != nil {
			return AbortWithMetaMessage(c, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
		}
		data, tag, err := handler(c, &req)
		if err != nil {
			return AbortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
		}
		Respond(c, data, tag)
		return nil
	})
	defaultGroup.Options(relativePath, func(c *fiber.Ctx) error {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			return AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
		}
		c.Set("Allow", "POST, OPTIONS")
		c.Set("Content-Type", web.ContentTypeMetaMessage)
		return c.Status(http.StatusOK).Send(encoded)
	})
}

func PUT[T any](relativePath string, handler Handler[T]) {
	defaultGroup.Put(relativePath, func(c *fiber.Ctx) error {
		var req T
		if err := bind(c, &req); err != nil {
			return AbortWithMetaMessage(c, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
		}
		data, tag, err := handler(c, &req)
		if err != nil {
			return AbortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
		}
		Respond(c, data, tag)
		return nil
	})
	defaultGroup.Options(relativePath, func(c *fiber.Ctx) error {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			return AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
		}
		c.Set("Allow", "PUT, OPTIONS")
		c.Set("Content-Type", web.ContentTypeMetaMessage)
		return c.Status(http.StatusOK).Send(encoded)
	})
}

func PATCH[T any](relativePath string, handler Handler[T]) {
	defaultGroup.Patch(relativePath, func(c *fiber.Ctx) error {
		var req T
		if err := bind(c, &req); err != nil {
			return AbortWithMetaMessage(c, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
		}
		data, tag, err := handler(c, &req)
		if err != nil {
			return AbortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
		}
		Respond(c, data, tag)
		return nil
	})
	defaultGroup.Options(relativePath, func(c *fiber.Ctx) error {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			return AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
		}
		c.Set("Allow", "PATCH, OPTIONS")
		c.Set("Content-Type", web.ContentTypeMetaMessage)
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

func Respond(c *fiber.Ctx, data any, tag string) {
	c.Locals("mm_response", data)
	c.Locals("mm_tag", tag)
	err := RespondFromLocals(c)
	if err != nil {
		c.Locals("mm_error", err)
	}
}

func RespondWithStatus(c *fiber.Ctx, code int, data any, tag string) {
	c.Locals("mm_response", data)
	c.Locals("mm_status", code)
	c.Locals("mm_tag", tag)
}

func AbortWithMetaMessage(c *fiber.Ctx, code int, obj any) error {
	encoded, err := mm.EncodeFromValue(obj, "")
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("encode error")
	}
	c.Set("Content-Type", web.ContentTypeMetaMessage)
	return c.Status(code).Send(encoded)
}

func RespondFromLocals(c *fiber.Ctx) error {
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
	c.Set("Content-Type", web.ContentTypeMetaMessage)
	return c.Status(status).Send(encoded)
}
