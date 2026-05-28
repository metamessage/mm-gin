package mmgin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	mm "github.com/metamessage/metamessage"
	"github.com/metamessage/mm-web-go/web"
)

// BindingError represents a field-level binding error.
type BindingError struct {
	Field   string
	Message string
	Code    string
}

// ValidationErrors is a collection of binding errors.
type ValidationErrors struct {
	Errors []BindingError
}

func (v ValidationErrors) Error() string {
	var msgs []string
	for _, e := range v.Errors {
		msgs = append(msgs, e.Field+": "+e.Message)
	}
	return strings.Join(msgs, "; ")
}

// ShouldBind attempts to bind request data to the struct, returning an error on failure.
func ShouldBind(c *gin.Context, obj any) error {
	return Bind(c, obj)
}

// MustBind binds request data and automatically returns a 400 error response on failure.
func MustBind(c *gin.Context, obj any) error {
	if err := Bind(c, obj); err != nil {
		AbortWithMetaMessage(c, http.StatusBadRequest, mmError{
			Error: "binding failed: " + err.Error(),
		})
		return err
	}
	return nil
}

// ShouldBindWithTag attempts to bind using the specified mm tag.
func ShouldBindWithTag(c *gin.Context, obj any, tag string) error {
	return BindWithTag(c, obj, tag)
}

// MustBindWithTag binds using the specified mm tag, returning an error response on failure.
func MustBindWithTag(c *gin.Context, obj any, tag string) error {
	if err := BindWithTag(c, obj, tag); err != nil {
		AbortWithMetaMessage(c, http.StatusBadRequest, mmError{
			Error: "binding failed0: " + err.Error(),
		})
		return err
	}
	return nil
}

// BindQuery binds query parameters to the struct.
// Converts query params to JSONC format and parses via MetaMessage.
func BindQuery(c *gin.Context, obj any) error {
	// Convert query params to map
	queryMap := make(map[string]any)
	for key, values := range c.Request.URL.Query() {
		if len(values) == 1 {
			queryMap[key] = values[0]
		} else {
			queryMap[key] = values
		}
	}

	// Encode map via MetaMessage and decode into target struct
	data, err := mm.EncodeFromValue(queryMap, "")
	if err != nil {
		return err
	}

	return mm.DecodeToValue(data, obj)
}

// BindHeader binds request headers to the struct.
func BindHeader(c *gin.Context, obj any) error {
	headerMap := make(map[string]any)
	for key, values := range c.Request.Header {
		if len(values) == 1 {
			headerMap[key] = values[0]
		} else {
			headerMap[key] = values
		}
	}

	data, err := mm.EncodeFromValue(headerMap, "")
	if err != nil {
		return err
	}

	return mm.DecodeToValue(data, obj)
}

// BindUri binds URI parameters to the struct.
func BindUri(c *gin.Context, obj any) error {
	uriMap := make(map[string]any)
	for i, param := range c.Params {
		uriMap[fmt.Sprintf("p%d", i)] = param.Value
		// Also use param name as key
		if param.Key != "" {
			uriMap[param.Key] = param.Value
		}
	}

	data, err := mm.EncodeFromValue(uriMap, "")
	if err != nil {
		return err
	}

	return mm.DecodeToValue(data, obj)
}

// AutoBind automatically selects binding method based on request content.
// Priority: URI params > query params > request body.
func AutoBind(c *gin.Context, obj any) error {
	// Bind URI params first
	if len(c.Params) > 0 {
		if err := BindUri(c, obj); err != nil {
			return err
		}
	}

	// Bind query params next (overrides URI params with same field name)
	if len(c.Request.URL.Query()) > 0 {
		if err := bindQueryToExisting(c, obj); err != nil {
			return err
		}
	}

	// Bind request body last (overrides previous fields with same name)
	if c.Request.Method != http.MethodGet &&
		c.Request.Method != http.MethodHead &&
		c.Request.Method != http.MethodDelete {
		if err := bindBodyToExisting(c, obj); err != nil {
			return err
		}
	}

	return nil
}

// bindQueryToExisting binds query params into an already-populated object.
func bindQueryToExisting(c *gin.Context, obj any) error {
	queryMap := make(map[string]any)
	for key, values := range c.Request.URL.Query() {
		if len(values) == 1 {
			queryMap[key] = values[0]
		} else {
			queryMap[key] = values
		}
	}

	return mergeIntoObject(obj, queryMap)
}

// bindBodyToExisting binds the request body into an already-populated object.
func bindBodyToExisting(c *gin.Context, obj any) error {
	body, exists := c.Get("mm_raw_body")
	if !exists {
		return nil
	}

	data := body.([]byte)
	if len(data) == 0 {
		return nil
	}

	formatVal, _ := c.Get("mm_format")
	format, _ := formatVal.(FormatType)

	var tempMap map[string]any
	switch format {
	case FormatMetaMessage:
		if err := mm.DecodeToValue(data, &tempMap); err != nil {
			return err
		}
	default:
		if err := mm.JsoncToValue(string(data), &tempMap); err != nil {
			return err
		}
	}

	return mergeIntoObject(obj, tempMap)
}

// mergeIntoObject merges a map into the target object.
func mergeIntoObject(obj any, data map[string]any) error {
	// Create a temporary struct to hold current object data
	objData, err := mm.EncodeFromValue(obj, "")
	if err != nil {
		return err
	}

	var objMap map[string]any
	if err := mm.DecodeToValue(objData, &objMap); err != nil {
		return err
	}

	// Merge data
	for key, value := range data {
		objMap[key] = value
	}

	// Re-encode and bind back to object
	mergedData, err := mm.EncodeFromValue(objMap, "")
	if err != nil {
		return err
	}

	return mm.DecodeToValue(mergedData, obj)
}

// Validator is the interface for custom data validation.
type Validator interface {
	Validate() error
}

// Validate performs custom validation.
// If the object implements the Validator interface, calls its Validate method.
func Validate(obj any) error {
	if v, ok := obj.(Validator); ok {
		return v.Validate()
	}
	return nil
}

// BindAndValidate binds and validates data.
func BindAndValidate(c *gin.Context, obj any) error {
	if err := Bind(c, obj); err != nil {
		return err
	}
	return Validate(obj)
}

// MustBindAndValidate binds and validates data, automatically returning an error response on failure.
func MustBindAndValidate[T any](c *gin.Context, obj *T) error {
	if err := Bind(c, obj); err != nil {
		AbortWithMetaMessage(c, http.StatusBadRequest, mmError{
			Error: "binding failed1: " + err.Error(),
		})
		return err
	}

	if err := Validate(obj); err != nil {
		AbortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{
			Error: "validation failed: " + err.Error(),
		})
		return err
	}

	return nil
}

// SetMMResponse sets a MetaMessage response (compatible with gin's JSON method style).
func SetMMResponse(c *gin.Context, code int, obj any) {
	c.Set("mm_response", obj)
	c.Status(code)
}

// JSONC returns a JSONC-format response.
func JSONC(c *gin.Context, code int, obj any) {
	jsoncStr, err := mm.ValueToJsonc(obj, "")
	if err != nil {
		AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{
			Error: "failed to encode response",
		})
		return
	}
	c.Data(code, web.ContentTypeJSONC, []byte(jsoncStr))
}

// MetaMessage returns a MetaMessage binary-format response.
func MetaMessage(c *gin.Context, code int, obj any) {
	data, err := mm.EncodeFromValue(obj, "")
	if err != nil {
		AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{
			Error: "failed to encode response",
		})
		return
	}
	c.Data(code, web.ContentTypeMetaMessage, data)
}
