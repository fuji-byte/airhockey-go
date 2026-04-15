package main

import (
	"fmt"
	"log"
	"runtime"

	"net/http"
	_ "net/http/pprof"

	"main/controllers"
	"main/dto"
	"main/repositories"
	"main/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	runtime.SetMutexProfileFraction(1)
	go func() {
		log.Println("pprof サーバー起動: http://localhost:6060/debug/pprof/")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Println("pprof サーバー終了:", err)
		}
	}()
	users := make(map[string]*dto.User, 0)
	gameRooms := make(map[string]*dto.GameRoom, 0)
	userMemoryRepository := repositories.NewMemoryRepository(users, gameRooms)
	userMemoryService := services.NewMemoryService(userMemoryRepository)
	userController := controllers.NewMemoryController(userMemoryService)

	r := gin.Default()
	// r.Use(cors.Default())
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		// AllowOrigins:     []string{"http://www.touhobby.com"},
		AllowMethods:     []string{"GET", "POST", "PUT"},            // 単純メソッドのみ許可
		AllowHeaders:     []string{"Content-Type", "Authorization"}, // 必要なヘッダーのみ許可
		AllowCredentials: true,
		MaxAge:           86400, // プリフライトリクエストを24時間キャッシュ
	}))

	fmt.Println("ws://localhost:8080で起動")
	r.GET("/ws", userController.HandleWebSocket)
	// r.GET("/login", RedisSessionController.Login)
	r.Run(":8080")
}
