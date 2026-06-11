package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/metamessage/mm-web-go/client"
	server "github.com/metamessage/mm-web-go/mmvanilla"
)

// go run ./examples/vanilla/main.go
type User struct {
	ID       int64  `mm:"desc=User ID"`
	Name     string `mm:"desc=User name; min=1; max=50"`
	Email    string `mm:"type=email; desc=Email address; allow_empty"`
	Age      uint8  `mm:"desc=Age; min=0; max=150; allow_empty"`
	IsActive bool   `mm:"desc=Is active"`
}

type CreateUserRequest struct {
	Name  string `mm:"desc=User name; min=1; max=50"`
	Email string `mm:"type=email; desc=Email address"`
	Age   uint8  `mm:"desc=Age; min=0; max=150"`
}

type CreateUserRequest2 struct {
	Name  string `mm:"desc=User name; min=1; max=50"`
	Email string `mm:"type=email; desc=Email address"`
	Age   uint8  `mm:"desc=Age; min=0; max=150"`
}

type UpdateUserRequest struct {
	Name     *string `mm:"desc=User name"`
	Email    *string `mm:"type=email; desc=Email address"`
	Age      *uint8  `mm:"desc=Age"`
	IsActive *bool   `mm:"desc=Is active"`
}

type ListUsersResponse struct {
	Total int64  `mm:"desc=Total"`
	Users []User `mm:"desc=User list"`
}

type APIResponse struct {
	Code    int    `mm:"desc=Code; allow_empty"`
	Message string `mm:"desc=Message; allow_empty"`
	Data    *User  `mm:"desc=Data; allow_empty"`
}

type HealthResponse struct {
	Status string `mm:"desc=Status"`
}

type ErrorResponse struct {
	Error string `mm:"desc=Error message"`
}

var users = []User{
	{ID: 1, Name: "Alice", Email: "alice@example.com", Age: 25, IsActive: true},
	{ID: 2, Name: "Bob", Email: "bob@example.com", Age: 30, IsActive: true},
	{ID: 3, Name: "Charlie", Email: "charlie@example.com", Age: 35, IsActive: false},
}

func listUsers(r *http.Request, _ *any) (any, string, error) {
	return ListUsersResponse{
		Total: int64(len(users)),
		Users: users,
	}, "", nil
}

func getUser(r *http.Request, u *User) (any, string, error) {
	fmt.Printf("url: %v\n", r.RequestURI)
	fmt.Printf("user: %+v\n", *u)
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	for _, u := range users {
		if u.ID == id {
			return APIResponse{Code: 0, Message: "success", Data: &u}, "", nil
		}
	}
	return nil, "", fmt.Errorf("user not found")
}

func createUser(r *http.Request, req *CreateUserRequest) (any, string, error) {
	newUser := User{
		ID:       int64(len(users) + 1),
		Name:     req.Name,
		Email:    req.Email,
		Age:      req.Age,
		IsActive: true,
	}
	users = append(users, newUser)
	return APIResponse{Code: 0, Message: "user created", Data: &newUser}, "", nil
}

func updateUser(r *http.Request, req *UpdateUserRequest) (any, string, error) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	for i, u := range users {
		if u.ID == id {
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

func deleteUser(r *http.Request, _ *any) (any, string, error) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	for i, u := range users {
		if u.ID == id {
			users = append(users[:i], users[i+1:]...)
			return APIResponse{Code: 0, Message: "user deleted"}, "", nil
		}
	}
	return nil, "", fmt.Errorf("user not found")
}

func healthCheck(r *http.Request, _ *any) (any, string, error) {
	return HealthResponse{Status: "ok"}, "", nil
}

func runTestsWithPort(port string) {
	baseURL := fmt.Sprintf("http://localhost:%s", port)

	fmt.Println(repeat("=", 100))
	fmt.Println("[Test] CRUD with MetaMessage binary protocol (net/http)...")
	fmt.Println(repeat("=", 100))

	client.SetDefaultClient(baseURL, true)

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

	fmt.Println(repeat("=", 100))
	fmt.Println("\n  [GET] /api/v1/user/1")
	u := User{
		ID:   223,
		Name: "322",
	}

	resp2, err := client.GET[User, APIResponse]("/api/v1/user/1", &u)
	if err != nil {
		fmt.Printf("  [Error] %v\n", err)
		return
	}
	fmt.Printf("  [OK] Message: %s\n", resp2.Message)
	fmt.Printf("     User: %+v\n", resp2.Data)

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

	fmt.Println(repeat("=", 100))
	fmt.Println("\n  [PUT] /api/v1/user/update/1")
	name := "Alice Updated"
	updateReq := &UpdateUserRequest{Name: &name}
	resp4, err := client.PUT[UpdateUserRequest, APIResponse]("/api/v1/user/update/1", updateReq)
	if err != nil {
		fmt.Printf("  [Error] %v\n", err)
		return
	}
	fmt.Printf("  [OK] Message: %s\n", resp4.Message)
	fmt.Printf("     Updated User: %+v\n", resp4.Data)

	fmt.Println(repeat("=", 100))
	fmt.Println("\n  [DELETE] /api/v1/user/delete/3")
	resp5, err := client.DELETE[any, APIResponse]("/api/v1/user/delete/3", nil)
	if err != nil {
		fmt.Printf("  [Error] %v\n", err)
		return
	}
	fmt.Printf("  [OK] Message: %s\n", resp5.Message)

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

func main() {
	mux := http.NewServeMux()

	server.Init(mux, "/api/v1")

	server.GET("/users", listUsers)
	server.GET("/user/{id}", getUser)
	server.POST("/user/create", createUser)
	server.PUT("/user/update/{id}", updateUser)
	server.DELETE("/user/delete/{id}", deleteUser)
	server.GET("/health", healthCheck)

	go func() {
		addr := ":8080"
		log.Printf("Server starting on %s", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	runTestsWithPort("8080")

	fmt.Println("\n" + repeat("=", 100))
	fmt.Println("[Done] All tests completed!")
	fmt.Println(repeat("=", 100))

	fmt.Println("\nPress Ctrl+C to exit...")
	select {}
}
