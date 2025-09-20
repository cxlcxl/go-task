package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"task-executor/config"
	"task-executor/database"
	"task-executor/executor"
	"task-executor/handlers"
	"task-executor/jobs"
	"task-executor/logger"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志
	if err = logger.Init(cfg.XXLJob.LogPath); err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}

	logger.Info("启动 Go 执行器...")
	logger.Info("服务器ID: %s", cfg.Executor.ServerID)
	logger.Info("配置: %+v", cfg)

	// 初始化数据库连接
	var db *database.Database
	var manager *jobs.Manager
	// 创建 PostgreSQL 数据库连接配置
	db, err = database.NewDatabase(cfg.PostgreSQL)
	if err != nil {
		logger.Error("连接PostgreSQL数据库失败: %v", err)
		os.Exit(1)
	}

	// 创建 River 队列管理器
	manager, err = jobs.NewManager(cfg, db, cfg.Executor.ServerID)
	if err != nil {
		logger.Error("初始化River管理器失败: %v", err)
		os.Exit(1)
	}

	// 启动 River 系统
	if err = manager.Start(); err != nil {
		logger.Error("启动River管理器失败: %v", err)
		os.Exit(1)
	}

	logger.Info("队列系统启动成功")
	// 创建XXL-JOB执行器
	exec := executor.NewExecutor(cfg)

	// 注册XXL-JOB任务处理器
	executor.RegisterXXLJobHandlers(exec)

	// 启动执行器（注册和心跳）
	if err = exec.Start(); err != nil {
		logger.Error("启动执行器失败: %v", err)
		os.Exit(1)
	}

	// 设置HTTP服务器
	httpHandler := handlers.NewHTTPHandler(exec)
	setupHTTPRoutes(httpHandler)

	// 启动HTTP服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Info("启动HTTP服务器: %s", addr)

	go func() {
		if err = http.ListenAndServe(addr, nil); err != nil {
			logger.Error("HTTP服务器启动失败: %v", err)
			os.Exit(1)
		}
	}()

	// 等待关闭信号
	waitForShutdown(manager, db)
}

func registerRiverTaskHandlers(riverManager *jobs.RiverManager) {
	// 注册队列任务处理器到River
	riverManager.RegisterTaskHandler("email", jobs.NewEmailTaskHandler())
	riverManager.RegisterTaskHandler("data_sync", jobs.NewDataSyncTaskHandler())
	riverManager.RegisterTaskHandler("file_process", jobs.NewFileProcessTaskHandler())
	riverManager.RegisterTaskHandler("report_generate", jobs.NewReportGenerateTaskHandler())
	riverManager.RegisterTaskHandler("notification", jobs.NewNotificationTaskHandler())

	logger.Info("所有River任务处理器注册成功")
}

func setupHTTPRoutes(handler *handlers.HTTPHandler) {
	http.HandleFunc("/beat", handler.Beat)
	http.HandleFunc("/idleBeat", handler.IdleBeat)
	http.HandleFunc("/run", handler.Run)
	http.HandleFunc("/kill", handler.Kill)
	http.HandleFunc("/log", handler.Log)

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	logger.Info("HTTP路由配置完成")
}

func waitForShutdown(manager *jobs.Manager, db *database.Database) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭执行器...")

	// 关闭River管理器（优雅关闭）
	if manager != nil {
		manager.Stop()
	}

	// 关闭数据库连接
	if db != nil {
		db.Close()
	}

	logger.Info("执行器已停止")
}
