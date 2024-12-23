package main

import (
	"net/http"

	"ip-geo/internal/api/handler"
	"ip-geo/internal/database"
	"ip-geo/internal/logger"
)

func main() {
	// 初始化数据库
	if err := database.InitializeDB(); err != nil {
		logger.Fatal("初始化数据库失败: %v", err)
	}
	logger.Info("数据库初始化成功")

	// 确保在程序退出时关闭数据库连接
	defer func() {
		db := database.GetInstance()
		db.Close()
	}()

	// 创建路由
	mux := http.NewServeMux()

	// 注册路由处理器
	ipHandler := handler.NewIPHandler()
	mux.HandleFunc("GET /api/ip/current", ipHandler.HandleCurrentIP)
	mux.HandleFunc("GET /api/ip/query/{ip}", ipHandler.HandleQueryIP)

	// 启动服务器
	addr := ":8080"
	logger.Info("服务器启动在 %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Fatal("服务器启动失败: %v", err)
	}
}
