package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/metamessage/mm-web-go/client"
	server "github.com/metamessage/mm-web-go/mmgin"
)

// go run ./examples/gin/main.go

// User represents a user entity.
// Native Go types are auto-inferred; no type tag needed.
// Special types (e.g., email) require explicit type=email tag.
// Avoid using json tags simultaneously; mm handles encoding automatically.
type User struct {
	ID       int64  `mm:"desc=用戶ID"`
	Name     string `mm:"desc=用戶名稱; min=1; max=50"`
	Email    string `mm:"type=email; desc=電子郵箱; allow_empty"`
	Age      uint8  `mm:"desc=年齡; min=0; max=150; allow_empty"`
	IsActive bool   `mm:"desc=是否激活"`
}

// CreateUserRequest is the request struct for creating a user.
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

// UpdateUserRequest is the request struct for updating a user.
type UpdateUserRequest struct {
	Name     *string `mm:"desc=用戶名稱"`
	Email    *string `mm:"type=email; desc=電子郵箱"`
	Age      *uint8  `mm:"desc=年齡"`
	IsActive *bool   `mm:"desc=是否激活"`
}

// ListUsersResponse is the response struct for listing users.
type ListUsersResponse struct {
	Total int64  `mm:"desc=總數"`
	Users []User `mm:"desc=用戶列表"`
}

// APIResponse is a generic API response wrapper.
type APIResponse struct {
	Code    int    `mm:"desc=狀態碼; allow_empty"`
	Message string `mm:"desc=消息; allow_empty"`
	Data    *User  `mm:"desc=數據; allow_empty"`
}

// HealthResponse is the response struct for health check.
type HealthResponse struct {
	Status string `mm:"desc=狀態"`
}

// ErrorResponse is the response struct for errors.
type ErrorResponse struct {
	Error string `mm:"desc=錯誤信息"`
}

// findAvailablePort finds an available port starting from the given port number.
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

// ============ Server ============

func runServer(port string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// One-line initialization: middleware + route group
	// After Init, all route methods use server.GET/POST/PUT/DELETE uniformly.
	server.Init(r, "/api/v1")

	// API routes
	// GET/DELETE generic functions auto-bind query params and register OPTIONS schema discovery.
	// POST/PUT generic functions auto-bind request bodies and register OPTIONS schema discovery.
	{
		// Data endpoints
		server.GET("/users", listUsers)
		server.GET("/user/:id", getUser)
		server.POST("/user/create", createUser)
		server.PUT("/user/update/:id", updateUser)
		server.DELETE("/user/delete/:id", deleteUser)
	}

	// Health check
	server.GET("/health", healthCheck)

	go func() {
		addr := ":" + port
		log.Printf("Server starting on %s", addr)
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

func listUsers(c *gin.Context, req *any) (any, string, error) {
	return ListUsersResponse{
		Total: int64(len(users)),
		Users: users,
	}, "", nil
}

func getUser(c *gin.Context, req *any) (any, string, error) {
	id := c.Param("id")
	for _, u := range users {
		if fmt.Sprintf("%d", u.ID) == id {
			return APIResponse{Code: 0, Message: "success", Data: &u}, "", nil
		}
	}
	return nil, "", fmt.Errorf("user not found")
}

func createUser(c *gin.Context, req *CreateUserRequest) (any, string, error) {
	newUser := User{
		ID:       int64(len(users) + 1),
		Name:     req.Name,
		Email:    req.Email,
		Age:      req.Age,
		IsActive: true,
	}
	users = append(users, newUser)

	return APIResponse{
		Code:    0,
		Message: "user created",
		Data:    &newUser,
	}, "", nil
}

func updateUser(c *gin.Context, req *UpdateUserRequest) (any, string, error) {
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

			return APIResponse{Code: 0, Message: "user updated", Data: &users[i]}, "", nil
		}
	}
	return nil, "", fmt.Errorf("user not found")
}

func deleteUser(c *gin.Context, req *any) (any, string, error) {
	id := c.Param("id")
	for i, u := range users {
		if fmt.Sprintf("%d", u.ID) == id {
			users = append(users[:i], users[i+1:]...)
			return APIResponse{Code: 0, Message: "user deleted", Data: nil}, "", nil
		}
	}
	return nil, "", fmt.Errorf("user not found")
}

func healthCheck(c *gin.Context, req *any) (any, string, error) {
	return HealthResponse{Status: "ok"}, "", nil
}

// ============ Test / Client Example ============

func runTestsWithPort(port string) {
	baseURL := fmt.Sprintf("http://localhost:%s", port)

	fmt.Println(repeat("=", 100))
	fmt.Println("[Test] CRUD with MetaMessage binary protocol...")
	fmt.Println(repeat("=", 100))

	client.SetDefaultClient(baseURL, true) // Set global default client for convenience

	// List users
	fmt.Println("\n  [GET] /api/v1/users")
	resp, err := client.GET[any, ListUsersResponse]("/api/v1/users", nil)
	if err != nil {
		fmt.Printf("  [Error] %v\n", err)
		return
	}
	fmt.Printf("  [OK] Total: %d\n", resp.Total)
	for _, u := range resp.Users {
		fmt.Printf("     - ID: %d, Name: %s, Email: %s, Age: %d\n", u.ID, u.Name, u.Email, u.Age)
	}

	// Get single user
	fmt.Println(repeat("=", 100))
	fmt.Println("\n  [GET] /api/v1/user/1")
	resp2, err := client.GET[any, APIResponse]("/api/v1/user/1", nil)
	if err != nil {
		fmt.Printf("  [Error] %v\n", err)
		return
	}
	fmt.Printf("  [OK] Message: %s\n", resp2.Message)
	fmt.Printf("     User: %+v\n", resp2.Data)

	// Create user
	fmt.Println(repeat("=", 100))
	fmt.Println("\n  [POST] /api/v1/user/create")
	createReq := &CreateUserRequest2{
		Name:  "David",
		Email: "david@example.com",
		Age:   28,
	}
	resp3, err := client.POST[CreateUserRequest2, APIResponse]("/api/v1/user/create", createReq)
	if err != nil {
		fmt.Printf("  [Error] %v\n", err)
		return
	}
	fmt.Printf("  [OK] Message: %s\n", resp3.Message)
	fmt.Printf("     New User: %+v\n", resp3.Data)

	// Update user
	fmt.Println(repeat("=", 100))
	fmt.Println("\n  [PUT] /api/v1/user/update/1")
	name := "Alice Updated"
	updateReq := &UpdateUserRequest{
		Name: &name,
	}
	resp4, err := client.PUT[UpdateUserRequest, APIResponse]("/api/v1/user/update/1", updateReq)
	if err != nil {
		fmt.Printf("  [Error] %v\n", err)
		return
	}
	fmt.Printf("  [OK] Message: %s\n", resp4.Message)
	fmt.Printf("     Updated User: %+v\n", resp4.Data)

	// Delete user
	fmt.Println(repeat("=", 100))
	fmt.Println("\n  [DELETE] /api/v1/user/delete/3")
	resp5, err := client.DELETE[any, APIResponse]("/api/v1/user/delete/3", nil)
	if err != nil {
		fmt.Printf("  [Error] %v\n", err)
		return
	}
	fmt.Printf("  [OK] Message: %s\n", resp5.Message)

	// Health check
	fmt.Println(repeat("=", 100))
	fmt.Println("\n  [HealthCheck] /api/v1/health")
	resp6, err := client.GET[any, HealthResponse]("/api/v1/health", nil)
	if err != nil {
		fmt.Printf("  [Error] %v\n", err)
		return
	}
	fmt.Printf("  [OK] Status: %s\n", resp6.Status)
}

func repeat(s string, n int) string {
	var sb strings.Builder
	sb.Grow(len(s) * n)

	for range n {
		sb.WriteString(s)
	}
	return sb.String()
}

// ============ Main ============

func main() {
	// Auto-find available port
	availablePort := findAvailablePort(8080)
	port := fmt.Sprintf("%d", availablePort)
	if availablePort != 8080 {
		fmt.Printf("[Warning] Port 8080 is in use, switching to port %s\n", port)
	}

	// Start server
	runServer(port)

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Run tests with the actual port
	runTestsWithPort(port)

	fmt.Println("\n" + repeat("=", 61))
	fmt.Println("[Done] All tests completed!")
	fmt.Println(repeat("=", 61))

	// Keep running for further testing
	fmt.Println("\nPress Ctrl+C to exit...")
	select {}
}
