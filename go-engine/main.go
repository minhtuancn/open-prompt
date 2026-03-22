package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/minhtuancn/open-prompt/go-engine/api"
	"github.com/minhtuancn/open-prompt/go-engine/config"
	"github.com/minhtuancn/open-prompt/go-engine/db"
)

func main() {
	// Đọc shared secret từ env (bắt buộc)
	secret := os.Getenv(config.SocketEnvKey)
	if secret == "" {
		log.Fatal("OP_SOCKET_SECRET is required")
	}

	// Khởi tạo database
	database, err := db.Open()
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Khởi động JSON-RPC server
	server, err := api.NewServer(secret, database)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	// Tạo listener trước để biết địa chỉ (cần thiết trên Windows để báo port)
	if err := server.PrepareListener(); err != nil {
		log.Fatalf("failed to prepare listener: %v", err)
	}

	go func() {
		if err := server.Listen(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Thông báo Tauri rằng engine đã ready (qua stdout)
	// Trên Windows: "ready:<port>" để Tauri biết TCP port cần connect
	// Trên Unix: "ready" (path socket đã biết từ config)
	if runtime.GOOS == "windows" {
		fmt.Printf("ready:%s\n", server.Addr())
	} else {
		fmt.Println("ready")
	}

	// Chờ signal để graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	server.Close()
}
