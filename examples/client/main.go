package main

import (
	"context"
	"fmt"
	"log"

	ginmm "github.com/metamessage/gin-mm"
)

// 獨立的客戶端示例
// 用於測試 mm-gin 服務端
func main() {
	ctx := context.Background()
	client := ginmm.NewClient("http://localhost:8080")

	// 測試健康檢查
	fmt.Println("=== 健康檢查 ===")
	if status, err := client.Health(ctx); err != nil {
		log.Printf("❌ 健康檢查失敗: %v", err)
	} else {
		fmt.Printf("✅ 健康: %s\n", status)
	}

	// 測試用戶列表
	fmt.Println("\n=== 獲取用戶列表 ===")
	users, err := client.ListUsers(ctx)
	if err != nil {
		log.Printf("❌ 獲取用戶列表失敗: %v", err)
		return
	}
	fmt.Printf("✅ 用戶總數: %d\n", users.Total)
	for _, u := range users.Users {
		fmt.Printf("   - %s (%s)\n", u.Name, u.Email)
	}

	// 測試獲取單個用戶
	fmt.Println("\n=== 獲取單個用戶 ===")
	user, err := client.GetUser(ctx, 1)
	if err != nil {
		log.Printf("❌ 獲取用戶失敗: %v", err)
		return
	}
	fmt.Printf("✅ 用戶: %s, Age: %d\n", user.Name, user.Age)

	// 測試創建用戶
	fmt.Println("\n=== 創建用戶 ===")
	newUser, err := client.CreateUser(ctx, &ginmm.CreateUserRequest{
		Name:  "Test User",
		Email: "test@example.com",
		Age:   25,
	})
	if err != nil {
		log.Printf("❌ 創建用戶失敗: %v", err)
		return
	}
	fmt.Printf("✅ 創建成功: %s\n", newUser.Message)

	// 測試更新用戶（使用指針類型部分更新）
	fmt.Println("\n=== 更新用戶 ===")
	name := "Updated Name"
	updatedUser, err := client.UpdateUser(ctx, 1, &ginmm.UpdateUserRequest{
		Name: &name,
	})
	if err != nil {
		log.Printf("❌ 更新用戶失敗: %v", err)
		return
	}
	fmt.Printf("✅ 更新成功: %s\n", updatedUser.Message)

	// 測試刪除用戶
	fmt.Println("\n=== 刪除用戶 ===")
	deleted, err := client.DeleteUser(ctx, 2)
	if err != nil {
		log.Printf("❌ 刪除用戶失敗: %v", err)
		return
	}
	fmt.Printf("✅ 刪除成功: %s\n", deleted.Message)
}
