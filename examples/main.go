package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	ginmm "github.com/metamessage/gin-mm"
)

// User 用戶結構體
type User struct {
	ID       int    `mm:"type=int64;desc=用戶ID" json:"id"`
	Name     string `mm:"type=str;desc=用戶名稱;min=1;max=50" json:"name"`
	Email    string `mm:"type=email;desc=電子郵箱" json:"email"`
	Age      int    `mm:"type=uint8;desc=年齡;min=0;max=150" json:"age"`
	IsActive bool   `mm:"type=bool;desc=是否激活" json:"is_active"`
}

// CreateUserRequest 創建用戶請求
type CreateUserRequest struct {
	Name  string `mm:"type=str;desc=用戶名稱;min=1;max=50" json:"name"`
	Email string `mm:"type=email;desc=電子郵箱" json:"email"`
	Age   int    `mm:"type=uint8;desc=年齡;min=0;max=150" json:"age"`
}

// Validate 自定義驗證
func (r *CreateUserRequest) Validate() error {
	if r.Age < 18 {
		return ginmm.ValidationErrors{
			Errors: []ginmm.BindingError{
				{Field: "age", Message: "用戶必須年滿18歲", Code: "age_too_young"},
			},
		}
	}
	return nil
}

// UpdateUserRequest 更新用戶請求
type UpdateUserRequest struct {
	Name     *string `mm:"type=str;desc=用戶名稱;nullable" json:"name,omitempty"`
	Email    *string `mm:"type=email;desc=電子郵箱;nullable" json:"email,omitempty"`
	Age      *int    `mm:"type=uint8;desc=年齡;nullable" json:"age,omitempty"`
	IsActive *bool   `mm:"type=bool;desc=是否激活;nullable" json:"is_active,omitempty"`
}

// ListUsersResponse 用戶列表響應
type ListUsersResponse struct {
	Total int    `mm:"type=int64;desc=總數" json:"total"`
	Users []User `mm:"type=slice;desc=用戶列表" json:"users"`
}

// APIResponse 通用 API 響應
type APIResponse struct {
	Code    int         `mm:"type=int32;desc=狀態碼" json:"code"`
	Message string      `mm:"type=str;desc=消息" json:"message"`
	Data    interface{} `mm:"desc=數據" json:"data"`
}

func main() {
	r := gin.Default()

	// 全局中間件
	r.Use(ginmm.MetaMessageDecoder(nil))
	r.Use(ginmm.MetaMessageEncoder(nil))

	// 路由組
	api := r.Group("/api/v1")
	{
		// 用戶相關路由
		api.GET("/users", listUsers)
		api.GET("/users/:id", getUser)
		api.POST("/users", createUser)
		api.PUT("/users/:id", updateUser)
		api.DELETE("/users/:id", deleteUser)
	}

	log.Println("Server starting on :8080")
	log.Println("Try these commands:")
	log.Println("  curl -X POST http://localhost:8080/api/v1/users \\")
	log.Println("    -H 'Content-Type: application/jsonc' \\")
	log.Println("    -d '{\"name\":\"Alice\",\"email\":\"alice@example.com\",\"age\":25}'")
	log.Println("")
	log.Println("  curl http://localhost:8080/api/v1/users/1")
	log.Println("")
	log.Println("  curl http://localhost:8080/api/v1/users")

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

// listUsers 獲取用戶列表
func listUsers(c *gin.Context) {
	// 模擬數據
	users := []User{
		{ID: 1, Name: "Alice", Email: "alice@example.com", Age: 25, IsActive: true},
		{ID: 2, Name: "Bob", Email: "bob@example.com", Age: 30, IsActive: true},
		{ID: 3, Name: "Charlie", Email: "charlie@example.com", Age: 35, IsActive: false},
	}

	response := ListUsersResponse{
		Total: len(users),
		Users: users,
	}

	ginmm.Respond(c, response)
}

// getUser 獲取單個用戶
func getUser(c *gin.Context) {
	id := c.Param("id")

	// 模擬數據
	user := User{
		ID:       1,
		Name:     "Alice",
		Email:    "alice@example.com",
		Age:      25,
		IsActive: true,
	}

	// 使用通用響應格式
	response := APIResponse{
		Code:    0,
		Message: "success",
		Data:    user,
	}

	ginmm.Respond(c, response)
}

// createUser 創建用戶
func createUser(c *gin.Context) {
	var req CreateUserRequest

	// 使用 MustBindAndValidate 自動綁定和驗證
	if err := ginmm.MustBindAndValidate(c, &req); err != nil {
		return // 錯誤響應已由 MustBindAndValidate 處理
	}

	// 創建用戶（模擬）
	user := User{
		ID:       1,
		Name:     req.Name,
		Email:    req.Email,
		Age:      req.Age,
		IsActive: true,
	}

	response := APIResponse{
		Code:    0,
		Message: "user created",
		Data:    user,
	}

	ginmm.RespondWithStatus(c, http.StatusCreated, response)
}

// updateUser 更新用戶
func updateUser(c *gin.Context) {
	id := c.Param("id")

	var req UpdateUserRequest
	if err := ginmm.MustBind(c, &req); err != nil {
		return
	}

	// 模擬更新
	user := User{
		ID:       1,
		Name:     "Alice",
		Email:    "alice@example.com",
		Age:      25,
		IsActive: true,
	}

	// 應用更新
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Age != nil {
		user.Age = *req.Age
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	response := APIResponse{
		Code:    0,
		Message: "user updated",
		Data:    user,
	}

	ginmm.Respond(c, response)
}

// deleteUser 刪除用戶
func deleteUser(c *gin.Context) {
	id := c.Param("id")

	response := APIResponse{
		Code:    0,
		Message: "user deleted",
		Data:    gin.H{"id": id},
	}

	ginmm.Respond(c, response)
}
