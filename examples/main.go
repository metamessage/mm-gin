package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	mmgin "github.com/metamessage/mm-gin"
	"github.com/metamessage/mm-gin/client"
)

// ============ 共享類型 ============

// User 用戶結構體
// Go 原生類型可自動推斷，不需 type 標籤
// 特殊類型（如 email）需手動 type=email
// 不建議同時使用 json 標籤，mm 會自動處理
type User struct {
	ID       int64  `mm:"desc=用戶ID"`
	Name     string `mm:"desc=用戶名稱; min=1; max=50"`
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

type CreateUserRequest2 struct {
	Name  string `mm:"desc=用戶名稱; min=1; max=50"`
	Email string `mm:"type=email; desc=電子郵箱"`
	Age   uint8  `mm:"desc=年齡; min=0; max=150"`
}

// Validate 自定義驗證
// func (r *CreateUserRequest) Validate() error {
// 	if r.Age < 18 {
// 		return fmt.Errorf("用戶必須年滿18歲")
// 	}
// 	return nil
// }

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

// ============ 測試示例 ============

func runTestsWithPort(port string) {
	baseURL := fmt.Sprintf("http://localhost:%s", port)

	fmt.Println("\n" + "=" + repeat("=", 60))
	fmt.Println("🧪 使用 MetaMessage 二進制協議測試 CRUD...")
	fmt.Println(repeat("=", 61) + "[]")

	client.SetDefaultClient(baseURL) // 設置全局默認客戶端，方便直接使用 mmgin.GET/POST 等函數

	// 列出用戶
	fmt.Println("\n  📋 GET /api/v1/users")
	resp, err := client.GET[any, ListUsersResponse]("/api/v1/users", nil)
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
	resp2, err := client.GET[any, APIResponse]("/api/v1/users/1", nil)
	if err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
		return
	}
	fmt.Printf("  ✅ 成功! Message: %s\n", resp2.Message)
	fmt.Printf("     User: %+v\n", resp2.Data)

	// 創建用戶
	fmt.Println("\n  📋 POST /api/v1/users")
	createReq := CreateUserRequest2{
		Name:  "David",
		Email: "david@example.com",
		Age:   28,
	}
	resp3, err := client.POST[CreateUserRequest2, APIResponse]("/api/v1/users", createReq)
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
	resp4, err := client.PUT[UpdateUserRequest, APIResponse]("/api/v1/users/1", updateReq)
	if err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
		return
	}
	fmt.Printf("  ✅ 成功! Message: %s\n", resp4.Message)
	fmt.Printf("     Updated User: %+v\n", resp4.Data)

	// 刪除用戶
	fmt.Println("\n  📋 DELETE /api/v1/users/3")
	resp5, err := client.DELETE[any, APIResponse]("/api/v1/users/3", nil)
	if err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
		return
	}
	fmt.Printf("  ✅ 成功! Message: %s\n", resp5.Message)

	// 健康檢查
	fmt.Println("\n  📋 GET /api/v1/health")
	resp6, err := client.GET[any, APIResponse]("/api/v1/health", nil)
	if err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
		return
	}
	fmt.Printf("  ✅ 成功! Message: %s\n", resp6.Message)
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
