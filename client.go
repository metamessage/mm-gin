package ginmm

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	mm "github.com/metamessage/metamessage"
)

// User 用戶結構體
// Go 原生類型可自動推斷，不需 type 標籤
// 特殊類型（如 email）需手動 type=email
type User struct {
	ID       int64  `mm:"desc=用戶ID"`
	Name     string `mm:"desc=用戶名稱"`
	Email    string `mm:"type=email;desc=電子郵箱"`
	Age      uint8  `mm:"desc=年齡"`
	IsActive bool   `mm:"desc=是否激活"`
}

// CreateUserRequest 創建用戶請求
type CreateUserRequest struct {
	Name  string `mm:"desc=用戶名稱"`
	Email string `mm:"type=email;desc=電子郵箱"`
	Age   uint8  `mm:"desc=年齡"`
}

// UpdateUserRequest 更新用戶請求
type UpdateUserRequest struct {
	Name     *string `mm:"desc=用戶名稱;nullable"`
	Email    *string `mm:"type=email;desc=電子郵箱;nullable"`
	Age      *uint8  `mm:"desc=年齡;nullable"`
	IsActive *bool   `mm:"desc=是否激活;nullable"`
}

// ListUsersResponse 用戶列表響應
type ListUsersResponse struct {
	Total int64  `mm:"desc=總數"`
	Users []User `mm:"desc=用戶列表"`
}

// APIResponse 通用 API 響應
type APIResponse struct {
	Code    int         `mm:"desc=狀態碼"`
	Message string      `mm:"desc=消息"`
	Data    interface{} `mm:"desc=數據"`
}

// HealthResponse 健康檢查響應
type HealthResponse struct {
	Status string `mm:"desc=狀態"`
}

// ErrorResponse 錯誤響應
type ErrorResponse struct {
	Error string `mm:"desc=錯誤信息"`
}

// Client HTTP 客戶端
type Client struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// ClientOption 客戶端選項
type ClientOption func(*Client)

// WithTimeout 設置超時時間
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
		c.httpClient.Timeout = timeout
	}
}

// NewClient 創建客戶端
func NewClient(baseURL string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// doRequest 執行 HTTP 請求
// 請求和響應都使用 MetaMessage 二進制格式
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := mm.EncodeFromValue(body, "")
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	// 請求和響應都使用 MetaMessage 二進制格式
	req.Header.Set("Content-Type", ContentTypeMetaMessage)
	req.Header.Set("Accept", ContentTypeMetaMessage)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// Health 健康檢查
func (c *Client) Health(ctx context.Context) (string, error) {
	data, err := c.doRequest(ctx, "GET", "/health", nil)
	if err != nil {
		return "", err
	}

	var result HealthResponse
	if err := mm.DecodeToValue(data, &result); err != nil {
		return "", err
	}
	return result.Status, nil
}

// Options 發送 OPTIONS 請求獲取端點的請求結構（Schema 發現）
// 返回值為 MetaMessage 二進制數據，包含結構體的類型、約束和描述信息
// 示例:
//
//	var schema CreateUserRequest
//	client.Options(ctx, "/api/v1/users", &schema)
func (c *Client) Options(ctx context.Context, path string, out any) error {
	data, err := c.doRequest(ctx, "OPTIONS", path, nil)
	if err != nil {
		return err
	}
	return mm.DecodeToValue(data, out)
}

// ListUsers 獲取用戶列表
func (c *Client) ListUsers(ctx context.Context) (*ListUsersResponse, error) {
	body, err := c.doRequest(ctx, "GET", "/api/v1/users", nil)
	if err != nil {
		return nil, err
	}

	var result ListUsersResponse
	if err := mm.DecodeToValue(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetUser 獲取單個用戶
func (c *Client) GetUser(ctx context.Context, id int64) (*User, error) {
	body, err := c.doRequest(ctx, "GET", "/api/v1/users/"+itoa(id), nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := mm.DecodeToValue(body, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUser 創建用戶
func (c *Client) CreateUser(ctx context.Context, req *CreateUserRequest) (*APIResponse, error) {
	body, err := c.doRequest(ctx, "POST", "/api/v1/users", req)
	if err != nil {
		return nil, err
	}

	var result APIResponse
	if err := mm.DecodeToValue(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateUser 更新用戶
func (c *Client) UpdateUser(ctx context.Context, id int64, req *UpdateUserRequest) (*APIResponse, error) {
	body, err := c.doRequest(ctx, "PUT", "/api/v1/users/"+itoa(id), req)
	if err != nil {
		return nil, err
	}

	var result APIResponse
	if err := mm.DecodeToValue(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteUser 刪除用戶
func (c *Client) DeleteUser(ctx context.Context, id int64) (*APIResponse, error) {
	body, err := c.doRequest(ctx, "DELETE", "/api/v1/users/"+itoa(id), nil)
	if err != nil {
		return nil, err
	}

	var result APIResponse
	if err := mm.DecodeToValue(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// itoa 將 int64 轉換為字符串
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	return string(result)
}
