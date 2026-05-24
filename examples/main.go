package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	mm "github.com/metamessage/metamessage"
	mmgin "github.com/metamessage/mm-gin"
)

// ============ 共享類型 ============

// User 用戶結構體
// Go 原生類型可自動推斷，不需 type 標籤
// 特殊類型（如 email）需手動 type=email
// 不建議同時使用 json 標籤，mm 會自動處理
type User struct {
	ID       int64  `mm:"desc=用戶ID"`
	Name     string `mm:"desc=用戶名稱; min=1; max=50; allow_empty"`
	Email    string `mm:"type=email; desc=電子郵箱; allow_empty"`
	Age      uint8  `mm:"desc=年齡; min=0; max=150; allow_empty"`
	IsActive bool   `mm:"desc=是否激活"`
}

// CreateUserRequest 創建用戶請求
type CreateUserRequest struct {
	Name  string `mm:"desc=用戶名稱; min=1; max=50"`
	Email string `mm:"type=email; desc=電子郵箱"`
	Age   uint8  `mm:"desc=年齡; min=0; max=150"`
}

// Validate 自定義驗證
func (r *CreateUserRequest) Validate() error {
	if r.Age < 18 {
		return fmt.Errorf("用戶必須年滿18歲")
	}
	return nil
}

// UpdateUserRequest 更新用戶請求
type UpdateUserRequest struct {
	Name     *string `mm:"desc=用戶名稱"`
	Email    *string `mm:"type=email; desc=電子郵箱"`
	Age      *uint8  `mm:"desc=年齡"`
	IsActive *bool   `mm:"desc=是否激活"`
}

// ListUsersResponse 用戶列表響應
type ListUsersResponse struct {
	Total int64  `mm:"desc=總數"`
	Users []User `mm:"desc=用戶列表"`
}

// APIResponse 通用 API 響應
type APIResponse struct {
	Code    int    `mm:"desc=狀態碼; allow_empty"`
	Message string `mm:"desc=消息; allow_empty"`
	Data    *User  `mm:"desc=數據; allow_empty"`
}

// HealthResponse 健康檢查響應
type HealthResponse struct {
	Status string `mm:"desc=狀態"`
}

// ErrorResponse 錯誤響應
type ErrorResponse struct {
	Error string `mm:"desc=錯誤信息"`
}

// findAvailablePort 從指定端口開始查找可用端口
func findAvailablePort(startPort int) int {
	for port := startPort; port < startPort+100; port++ {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return port
		}
	}
	return startPort
}

// ============ 服務端 ============

func runServer(port string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// 一行初始化：中間件 + 路由分組
	// Init 後所有路由方法統一使用 mmgin.GET/POST/PUT/DELETE 等
	mmgin.Init(r, "/api/v1")

	// API 路由
	// POST/PUT 泛型函數自動綁定請求並註冊 OPTIONS
	{
		// 數據端點
		mmgin.GET("/users", listUsers)
		mmgin.GET("/users/:id", getUser)
		mmgin.POST("/users", createUser)
		mmgin.PUT("/users/:id", updateUser)
		mmgin.DELETE("/users/:id", deleteUser)
	}

	// 健康檢查
	r.GET("/health", func(c *gin.Context) {
		mmgin.Respond(c, HealthResponse{Status: "ok"}, "")
	})

	go func() {
		addr := ":" + port
		log.Printf("🚀 Server starting on %s", addr)
		if err := r.Run(addr); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	return r
}

var users = []User{
	{ID: 1, Name: "Alice", Email: "alice@example.com", Age: 25, IsActive: true},
	{ID: 2, Name: "Bob", Email: "bob@example.com", Age: 30, IsActive: true},
	{ID: 3, Name: "Charlie", Email: "charlie@example.com", Age: 35, IsActive: false},
}

func listUsers(c *gin.Context) {
	mmgin.Respond(c, ListUsersResponse{
		Total: int64(len(users)),
		Users: users,
	}, "desc=用戶列表響應")
}

func getUser(c *gin.Context) {
	id := c.Param("id")
	for _, u := range users {
		if fmt.Sprintf("%d", u.ID) == id {
			mmgin.Respond(c, APIResponse{Code: 0, Message: "success", Data: &u}, "")
			return
		}
	}
	mmgin.AbortWithMetaMessage(c, http.StatusNotFound, ErrorResponse{Error: "user not found"})
}

func createUser(c *gin.Context, req *CreateUserRequest) {
	newUser := User{
		ID:       int64(len(users) + 1),
		Name:     req.Name,
		Email:    req.Email,
		Age:      req.Age,
		IsActive: true,
	}
	users = append(users, newUser)

	mmgin.RespondWithStatus(c, http.StatusCreated, APIResponse{
		Code:    0,
		Message: "user created",
		Data:    &newUser,
	}, "")
}

func updateUser(c *gin.Context, req *UpdateUserRequest) {
	id := c.Param("id")

	for i, u := range users {
		if fmt.Sprintf("%d", u.ID) == id {
			if req.Name != nil {
				users[i].Name = *req.Name
			}
			if req.Email != nil {
				users[i].Email = *req.Email
			}
			if req.Age != nil {
				users[i].Age = *req.Age
			}
			if req.IsActive != nil {
				users[i].IsActive = *req.IsActive
			}

			mmgin.Respond(c, APIResponse{Code: 0, Message: "user updated", Data: &users[i]}, "")
			return
		}
	}
	mmgin.AbortWithMetaMessage(c, http.StatusNotFound, ErrorResponse{Error: "user not found"})
}

func deleteUser(c *gin.Context) {
	id := c.Param("id")
	for i, u := range users {
		if fmt.Sprintf("%d", u.ID) == id {
			users = append(users[:i], users[i+1:]...)
			mmgin.Respond(c, APIResponse{Code: 0, Message: "user deleted", Data: nil}, "")
			return
		}
	}
	mmgin.AbortWithMetaMessage(c, http.StatusNotFound, ErrorResponse{Error: "user not found"})
}

// ============ 客戶端 ============

// Client HTTP 客戶端
// 請求體發送 JSONC 格式，響應始終接收 MetaMessage 二進制格式
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient 創建客戶端
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// doRequest 執行 HTTP 請求
// 請求和響應都使用 MetaMessage 二進制格式
func (c *Client) doRequest(method, path string, body any) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := mm.EncodeFromValue(body, "")
		if err != nil {
			return nil, 0, fmt.Errorf("encode metamessage: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, 0, err
	}

	// 請求和響應都使用 MetaMessage 二進制格式
	req.Header.Set("Content-Type", "application/x-metamessage")
	req.Header.Set("Accept", "application/x-metamessage")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)

	return data, resp.StatusCode, err
}

// ListUsers 獲取用戶列表
func (c *Client) ListUsers(ctx context.Context) (*ListUsersResponse, error) {
	data, _, err := c.doRequest("GET", "/api/v1/users", nil)
	if err != nil {
		return nil, err
	}

	// 解碼 MetaMessage 二進制到結構體
	var result ListUsersResponse
	if err := mm.DecodeToValue(data, &result); err != nil {
		return nil, fmt.Errorf("decode metamessage: %w", err)
	}
	jsonc, err := mm.DecodeToJsonc(data)
	if err != nil {
		return nil, fmt.Errorf("decode metamessage: %w", err)
	}
	fmt.Println("res", jsonc)
	return &result, nil
}

// GetUser 獲取單個用戶
func (c *Client) GetUser(ctx context.Context, id int64) (*APIResponse, error) {
	data, _, err := c.doRequest("GET", fmt.Sprintf("/api/v1/users/%d", id), nil)
	if err != nil {
		return nil, err
	}

	var result APIResponse
	if err := mm.DecodeToValue(data, &result); err != nil {
		jsonc, _ := mm.DecodeToJsonc(data)
		fmt.Println("error", jsonc)
		return nil, fmt.Errorf("decode metamessage: %w", err)
	}
	return &result, nil
}

// CreateUser 創建用戶
func (c *Client) CreateUser(ctx context.Context, req CreateUserRequest) (*APIResponse, error) {
	data, _, err := c.doRequest("POST", "/api/v1/users", req)
	if err != nil {
		return nil, err
	}

	var result APIResponse
	if err := mm.DecodeToValue(data, &result); err != nil {
		jsonc, _ := mm.DecodeToJsonc(data)
		fmt.Println("error", jsonc)
		return nil, fmt.Errorf("decode metamessage: %w", err)
	}
	return &result, nil
}

// UpdateUser 更新用戶
func (c *Client) UpdateUser(ctx context.Context, id int64, req UpdateUserRequest) (*APIResponse, error) {
	data, _, err := c.doRequest("PUT", fmt.Sprintf("/api/v1/users/%d", id), req)
	if err != nil {
		return nil, err
	}

	var result APIResponse
	if err := mm.DecodeToValue(data, &result); err != nil {
		jsonc, _ := mm.DecodeToJsonc(data)
		fmt.Println("error", jsonc)
		return nil, fmt.Errorf("decode metamessage: %w", err)
	}
	return &result, nil
}

// DeleteUser 刪除用戶
func (c *Client) DeleteUser(ctx context.Context, id int64) (*APIResponse, error) {
	data, _, err := c.doRequest("DELETE", fmt.Sprintf("/api/v1/users/%d", id), nil)
	if err != nil {
		return nil, err
	}

	var result APIResponse
	if err := mm.DecodeToValue(data, &result); err != nil {
		jsonc, _ := mm.DecodeToJsonc(data)
		fmt.Println("error", jsonc)
		return nil, fmt.Errorf("decode metamessage: %w", err)
	}
	return &result, nil
}

// Health 健康檢查
func (c *Client) Health(ctx context.Context) (string, error) {
	data, _, err := c.doRequest("GET", "/health", nil)
	if err != nil {
		return "", err
	}

	var result HealthResponse
	if err := mm.DecodeToValue(data, &result); err != nil {
		return "", fmt.Errorf("decode metamessage: %w", err)
	}
	return result.Status, nil
}

// Schema 發送 OPTIONS 請求獲取端點的請求結構
// out 是目標結構體的指針，用於接收解碼後的結構信息
// 返回結構體的 JSONC 表示（可用於打印）
func (c *Client) Schema(ctx context.Context, method, path string, out any) (string, error) {
	data, _, err := c.doRequest("OPTIONS", path, nil)
	if err != nil {
		return "", err
	}

	if out != nil {
		if err := mm.DecodeToValue(data, out); err != nil {
			jsonc, _ := mm.DecodeToJsonc(data)
			fmt.Println("error", jsonc)
			return "", fmt.Errorf("decode metamessage: %w", err)
		}
	}

	// 解碼為 JSONC 用於打印
	jsoncStr, err := mm.DecodeToJsonc(data)
	if err != nil {
		jsonc, _ := mm.DecodeToJsonc(data)
		fmt.Println("error", jsonc)
		return "", fmt.Errorf("decode to jsonc: %w", err)
	}
	return jsoncStr, nil
}

// ============ 測試示例 ============

func runTestsWithPort(port string) {
	ctx := context.Background()
	baseURL := fmt.Sprintf("http://localhost:%s", port)

	fmt.Println("\n" + "=" + repeat("=", 60))
	fmt.Println("🧪 使用 MetaMessage 二進制協議測試 CRUD...")
	fmt.Println(repeat("=", 61) + "[]")

	client := NewClient(baseURL)

	// 列出用戶
	fmt.Println("\n  📋 GET /api/v1/users")
	resp, err := client.ListUsers(ctx)
	if err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
		return
	}
	fmt.Printf("  ✅ 成功! Total: %d\n", resp.Total)
	for _, u := range resp.Users {
		fmt.Printf("     - ID: %d, Name: %s, Email: %s, Age: %d\n", u.ID, u.Name, u.Email, u.Age)
	}

	// 獲取單個用戶
	fmt.Println("\n  📋 GET /api/v1/users/1")
	resp2, err := client.GetUser(ctx, 1)
	if err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
		return
	}
	fmt.Printf("  ✅ 成功! Message: %s\n", resp2.Message)
	fmt.Printf("     User: %+v\n", resp2.Data)

	// 創建用戶
	fmt.Println("\n  📋 POST /api/v1/users")
	createReq := CreateUserRequest{
		Name:  "David",
		Email: "david@example.com",
		Age:   28,
	}
	resp3, err := client.CreateUser(ctx, createReq)
	if err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
		return
	}
	fmt.Printf("  ✅ 成功! Message: %s\n", resp3.Message)
	fmt.Printf("     New User: %+v\n", resp3.Data)

	// 更新用戶
	fmt.Println("\n  📋 PUT /api/v1/users/1")
	name := "Alice Updated"
	updateReq := UpdateUserRequest{
		Name: &name,
	}
	resp4, err := client.UpdateUser(ctx, 1, updateReq)
	if err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
		return
	}
	fmt.Printf("  ✅ 成功! Message: %s\n", resp4.Message)
	fmt.Printf("     Updated User: %+v\n", resp4.Data)

	// Schema 發現測試
	fmt.Println("\n  📋 OPTIONS /api/v1/users (Schema 發現)")
	var createSchema CreateUserRequest
	createJSONC, err := client.Schema(ctx, "POST", "/api/v1/users", &createSchema)
	if err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ JSONC:\n%s\n", createJSONC)
	}

	fmt.Println("\n  📋 OPTIONS /api/v1/users/:id (Schema 發現)")
	var updateSchema UpdateUserRequest
	updateJSONC, err := client.Schema(ctx, "PUT", "/api/v1/users/1", &updateSchema)
	if err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ JSONC:\n%s\n", updateJSONC)
	}

	// 刪除用戶
	fmt.Println("\n  📋 DELETE /api/v1/users/3")
	resp5, err := client.DeleteUser(ctx, 3)
	if err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
		return
	}
	fmt.Printf("  ✅ 成功! Message: %s\n", resp5.Message)
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

// ============ 主函數 ============

func main() {
	// 自動查找可用端口
	availablePort := findAvailablePort(8080)
	port := fmt.Sprintf("%d", availablePort)
	if availablePort != 8080 {
		fmt.Printf("⚠️  端口 8080 已被佔用，自動切換到端口 %s\n", port)
	}

	// 啟動服務端
	runServer(port)

	// 等待服務端啟動
	time.Sleep(500 * time.Millisecond)

	// 使用實際端口運行測試
	runTestsWithPort(port)

	fmt.Println("\n" + repeat("=", 61))
	fmt.Println("✨ 所有測試完成!")
	fmt.Println(repeat("=", 61))

	// 保持運行以便繼續測試
	fmt.Println("\n按 Ctrl+C 退出...")
	select {}
}
