package mmgin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	mm "github.com/metamessage/metamessage"
)

// BindingError 綁定錯誤
type BindingError struct {
	Field   string
	Message string
	Code    string
}

// ValidationErrors 驗證錯誤集合
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

// ShouldBind 嘗試綁定請求數據到結構體，失敗返回錯誤
func ShouldBind(c *gin.Context, obj any) error {
	return Bind(c, obj)
}

// MustBind 綁定請求數據，失敗時自動返回 400 錯誤響應
func MustBind(c *gin.Context, obj any) error {
	if err := Bind(c, obj); err != nil {
		AbortWithMetaMessage(c, http.StatusBadRequest, mmError{
			Error: "binding failed: " + err.Error(),
		})
		return err
	}
	return nil
}

// ShouldBindWithTag 使用指定 tag 嘗試綁定
func ShouldBindWithTag(c *gin.Context, obj any, tag string) error {
	return BindWithTag(c, obj, tag)
}

// MustBindWithTag 使用指定 tag 綁定，失敗時自動返回錯誤
func MustBindWithTag(c *gin.Context, obj any, tag string) error {
	if err := BindWithTag(c, obj, tag); err != nil {
		AbortWithMetaMessage(c, http.StatusBadRequest, mmError{
			Error: "binding failed: " + err.Error(),
		})
		return err
	}
	return nil
}

// BindQuery 綁定查詢參數到結構體
// 將查詢參數轉換為 JSONC 格式後使用 metamessage 解析
func BindQuery(c *gin.Context, obj any) error {
	// 將查詢參數轉換為 map
	queryMap := make(map[string]any)
	for key, values := range c.Request.URL.Query() {
		if len(values) == 1 {
			queryMap[key] = values[0]
		} else {
			queryMap[key] = values
		}
	}

	// 使用 metamessage 將 map 轉換為節點再綁定
	data, err := mm.EncodeFromValue(queryMap, "")
	if err != nil {
		return err
	}

	return mm.DecodeToValue(data, obj)
}

// BindHeader 綁定請求頭到結構體
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

// BindUri 綁定 URI 參數到結構體
func BindUri(c *gin.Context, obj any) error {
	uriMap := make(map[string]any)
	for i, param := range c.Params {
		uriMap[fmt.Sprintf("p%d", i)] = param.Value
		// 同時用參數名作為 key
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

// AutoBind 自動根據請求內容選擇綁定方式
// 優先順序：URI 參數 > 查詢參數 > 請求體
func AutoBind(c *gin.Context, obj any) error {
	// 先綁定 URI 參數
	if len(c.Params) > 0 {
		if err := BindUri(c, obj); err != nil {
			return err
		}
	}

	// 再綁定查詢參數（會覆蓋 URI 參數中的同名字段）
	if len(c.Request.URL.Query()) > 0 {
		if err := bindQueryToExisting(c, obj); err != nil {
			return err
		}
	}

	// 最後綁定請求體（會覆蓋之前的同名字段）
	if c.Request.Method != http.MethodGet &&
		c.Request.Method != http.MethodHead &&
		c.Request.Method != http.MethodDelete {
		if err := bindBodyToExisting(c, obj); err != nil {
			return err
		}
	}

	return nil
}

// bindQueryToExisting 將查詢參數綁定到已存在的對象
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

// bindBodyToExisting 將請求體綁定到已存在的對象
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

// mergeIntoObject 將 map 合併到目標對象
func mergeIntoObject(obj any, data map[string]any) error {
	// 創建一個臨時結構來存儲當前對象的數據
	objData, err := mm.EncodeFromValue(obj, "")
	if err != nil {
		return err
	}

	var objMap map[string]any
	if err := mm.DecodeToValue(objData, &objMap); err != nil {
		return err
	}

	// 合併數據
	for key, value := range data {
		objMap[key] = value
	}

	// 重新編碼並綁回對象
	mergedData, err := mm.EncodeFromValue(objMap, "")
	if err != nil {
		return err
	}

	return mm.DecodeToValue(mergedData, obj)
}

// Validator 數據驗證器接口
type Validator interface {
	Validate() error
}

// Validate 執行自定義驗證
// 如果對象實現了 Validator 接口，則調用其 Validate 方法
func Validate(obj any) error {
	if v, ok := obj.(Validator); ok {
		return v.Validate()
	}
	return nil
}

// BindAndValidate 綁定並驗證數據
func BindAndValidate(c *gin.Context, obj any) error {
	if err := Bind(c, obj); err != nil {
		return err
	}
	return Validate(obj)
}

// MustBindAndValidate 綁定並驗證數據，失敗時自動返回錯誤響應
func MustBindAndValidate(c *gin.Context, obj any) error {
	if err := Bind(c, obj); err != nil {
		AbortWithMetaMessage(c, http.StatusBadRequest, mmError{
			Error: "binding failed: " + err.Error(),
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

// SetMMResponse 設置 MetaMessage 響應（兼容 gin 的 JSON 方法風格）
func SetMMResponse(c *gin.Context, code int, obj any) {
	c.Set("mm_response", obj)
	c.Status(code)
}

// JSONC 返回 JSONC 格式的響應
func JSONC(c *gin.Context, code int, obj any) {
	jsoncStr, err := mm.ValueToJsonc(obj, "")
	if err != nil {
		AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{
			Error: "failed to encode response",
		})
		return
	}
	c.Data(code, ContentTypeJSONC, []byte(jsoncStr))
}

// MetaMessage 返回 MetaMessage 二進制格式的響應
func MetaMessage(c *gin.Context, code int, obj any) {
	data, err := mm.EncodeFromValue(obj, "")
	if err != nil {
		AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{
			Error: "failed to encode response",
		})
		return
	}
	c.Data(code, ContentTypeMetaMessage, data)
}
