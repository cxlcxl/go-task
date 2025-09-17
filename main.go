package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"xxljob-go-executor/admin"
	"xxljob-go-executor/config"
	"xxljob-go-executor/database"
	"xxljob-go-executor/executor"
	"xxljob-go-executor/handlers"
	"xxljob-go-executor/jobs"
	"xxljob-go-executor/logger"
	"xxljob-go-executor/queue"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志
	if err := logger.Init(cfg.XXLJob.LogPath); err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}

	logger.Info("启动XXL-JOB Go执行器...")
	logger.Info("服务器ID: %s", cfg.Executor.ServerID)
	logger.Info("配置: %+v", cfg)

	// 初始化数据库连接
	var db *database.Database
	var queueManager *queue.QueueManager

	if cfg.Queue.Enable {
		// 创建多数据库连接配置
		dbConfigs := make(map[string]database.Config)
		for name, dbCfg := range cfg.Database {
			dbConfigs[name] = database.Config{
				Host:     dbCfg.Host,
				Port:     dbCfg.Port,
				Username: dbCfg.Username,
				Password: dbCfg.Password,
				Database: dbCfg.Database,
				Charset:  dbCfg.Charset,
			}
		}

		db, err = database.NewDatabase(dbConfigs)
		if err != nil {
			logger.Error("连接数据库失败: %v", err)
			os.Exit(1)
		}

		// 自动迁移数据库表
		// if err := db.AutoMigrate(); err != nil {
		// 	logger.Error("数据库表迁移失败: %v", err)
		// 	os.Exit(1)
		// }

		// 创建队列管理器
		queueManager = queue.NewQueueManager(db, cfg.Executor.ServerID)

		// 注册队列任务处理器
		registerQueueTaskHandlers(queueManager)

		// 启动队列管理器
		if err := queueManager.Start(); err != nil {
			logger.Error("启动队列管理器失败: %v", err)
			os.Exit(1)
		}
		logger.Info("队列管理器启动成功")
	}

	// 创建XXL-JOB执行器
	exec := executor.NewExecutor(cfg)

	// 注册XXL-JOB任务处理器
	registerXXLJobHandlers(exec)

	// 启动执行器（注册和心跳）
	if err := exec.Start(); err != nil {
		logger.Error("启动执行器失败: %v", err)
		os.Exit(1)
	}

	// 设置HTTP服务器
	httpHandler := handlers.NewHTTPHandler(exec)
	setupHTTPRoutes(httpHandler)

	// 设置管理接口
	if db != nil {
		adminHandler := admin.NewAdminHandler(db)
		setupAdminRoutes(adminHandler)
	}

	// 启动HTTP服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Info("启动HTTP服务器: %s", addr)

	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			logger.Error("HTTP服务器启动失败: %v", err)
			os.Exit(1)
		}
	}()

	// 等待关闭信号
	waitForShutdown(queueManager)
}

func registerXXLJobHandlers(exec *executor.Executor) {
	// 注册各种任务处理器
	exec.RegisterJobHandler("demoJobHandler", jobs.NewDemoJobHandler())
	exec.RegisterJobHandler("dataProcessJobHandler", jobs.NewDataProcessJobHandler())
	exec.RegisterJobHandler("emailJobHandler", jobs.NewEmailJobHandler())
	exec.RegisterJobHandler("reportJobHandler", jobs.NewReportJobHandler())
	exec.RegisterJobHandler("shardingJobHandler", jobs.NewShardingJobHandler())
	exec.RegisterJobHandler("failureJobHandler", jobs.NewFailureJobHandler())

	logger.Info("所有XXL-JOB任务处理器注册成功")
}

func registerQueueTaskHandlers(queueManager *queue.QueueManager) {
	// 注册队列任务处理器
	queueManager.RegisterTaskHandler("email", jobs.NewEmailTaskHandler())
	queueManager.RegisterTaskHandler("data_sync", jobs.NewDataSyncTaskHandler())
	queueManager.RegisterTaskHandler("file_process", jobs.NewFileProcessTaskHandler())
	queueManager.RegisterTaskHandler("report_generate", jobs.NewReportGenerateTaskHandler())
	queueManager.RegisterTaskHandler("notification", jobs.NewNotificationTaskHandler())

	logger.Info("所有队列任务处理器注册成功")
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

func setupAdminRoutes(handler *admin.AdminHandler) {
	// 队列管理接口
	http.HandleFunc("/admin/queue/create", handler.CreateQueueConfig)
	http.HandleFunc("/admin/queue/update", handler.UpdateQueueConfig)
	http.HandleFunc("/admin/queue/list", handler.GetQueueConfigs)
	http.HandleFunc("/admin/queue/delete", handler.DeleteQueueConfig)

	// 服务器管理接口
	http.HandleFunc("/admin/server/list", handler.GetServerInfos)

	// 任务管理接口
	http.HandleFunc("/admin/task/create", handler.CreateTask)

	logger.Info("管理接口路由配置完成")
}

func waitForShutdown(queueManager *queue.QueueManager) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭执行器...")

	// 关闭队列管理器
	if queueManager != nil {
		queueManager.Stop()
	}

	// TODO: 实现优雅关闭
	// - 停止接收新任务
	// - 等待正在执行的任务完成
	// - 从 XXL-JOB 管理台取消注册
	logger.Info("执行器已停止")
}
