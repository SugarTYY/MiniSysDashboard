package main

import (
	"MiniSysDashboard/internal/api"
	"MiniSysDashboard/internal/collector"
	"MiniSysDashboard/internal/model"
	"MiniSysDashboard/internal/storage"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 初始化配置
	config := model.DefaultConfig
	// TODO: 如果需要，从文件或环境变量加载配置

	// 2. 初始化存储
	// 确保 data 目录存在
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	dbPath := filepath.Join(dataDir, "metrics.db")
	store, err := storage.NewStorage(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// 3. 初始化采集器
	coll := collector.NewCollector(config)

	// 4. 初始化 API 处理器
	handler := api.NewHandler(coll, store)

	// 5. 设置路由
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 服务静态文件 (SPA)
	// 在生产环境 (Docker) 中，前端文件位于 ./static
	// 在开发环境中，我们可能不由 Go 提供服务，或者使用不同的路径。
	// 为了简单起见，我们检查 ./static 是否存在。
	staticDir := "./static"
	if _, err := os.Stat(staticDir); err == nil {
		r.Static("/assets", filepath.Join(staticDir, "assets"))
		r.StaticFile("/favicon.svg", filepath.Join(staticDir, "favicon.svg"))

		// SPA 的兜底路由
		r.NoRoute(func(c *gin.Context) {
			// 检查是否为 API 请求
			if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
				c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
				return
			}
			if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/sse" {
				c.Status(http.StatusNotFound)
				return
			}

			c.File(filepath.Join(staticDir, "index.html"))
		})
	}

	handler.RegisterRoutes(r)

	// 6. 启动后台采集
	handler.StartCollection(time.Duration(config.CollectionInterval) * time.Second)

	// 7. 启动服务器并支持优雅停机
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		log.Println("Server starting on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// 等待中断信号以优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}
