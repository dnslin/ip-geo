package main

import (
	"net/http"

	"ip-geo/internal/api/handler"
	"ip-geo/internal/database"
	"ip-geo/internal/downloader"
	"ip-geo/internal/logger"
)

func main() {
	// 确保MMDB文件存在
	if err := downloader.EnsureMMDBFiles(); err != nil {
		logger.Fatal("确保MMDB文件存在失败: %v", err)
	}

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
	// 简化路由
	mux.HandleFunc("GET /ip", ipHandler.HandleCurrentIP)      // 获取当前IP
	mux.HandleFunc("GET /ip/{ip}", ipHandler.HandleQueryIP)   // 查询指定IP

	// 启动服务器
	addr := ":8080"
	logger.Info("服务器启动在 %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Fatal("服务器启动失败: %v", err)
	}
}
