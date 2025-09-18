package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// TaskRequest 任务请求结构
type TaskRequest struct {
	JobID       int                    `json:"job_id"`
	JobParam    string                 `json:"job_param"`
	LogID       int                    `json:"log_id"`
	LogDateTime int64                  `json:"log_datetime"`
	TaskType    string                 `json:"task_type"`
	Version     string                 `json:"version"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// TaskResponse 任务响应结构
type TaskResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// BusinessHandler 业务处理器
type BusinessHandler struct {
	handlerName string
	version     string
}

func NewBusinessHandler(name, version string) *BusinessHandler {
	return &BusinessHandler{
		handlerName: name,
		version:     version,
	}
}

// 示例业务处理器实现
func (h *BusinessHandler) HandleDataExport(w http.ResponseWriter, r *http.Request) {
	h.handleTask(w, r, h.processDataExport)
}

func (h *BusinessHandler) HandleUserNotification(w http.ResponseWriter, r *http.Request) {
	h.handleTask(w, r, h.processUserNotification)
}

func (h *BusinessHandler) HandleReportGeneration(w http.ResponseWriter, r *http.Request) {
	h.handleTask(w, r, h.processReportGeneration)
}

func (h *BusinessHandler) handleTask(w http.ResponseWriter, r *http.Request, processor func(*TaskRequest) error) {
	if r.Method != http.MethodPost {
		h.writeErrorResponse(w, "Method not allowed", nil)
		return
	}

	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, "Invalid request body", err)
		return
	}

	log.Printf("Processing task: type=%s, job_id=%d", req.TaskType, req.JobID)

	// 处理任务
	if err := processor(&req); err != nil {
		h.writeErrorResponse(w, "Task processing failed", err)
		return
	}

	h.writeSuccessResponse(w, "Task completed successfully", nil)
}

// 具体业务处理逻辑
func (h *BusinessHandler) processDataExport(req *TaskRequest) error {
	log.Printf("Starting data export for job %d", req.JobID)

	// 解析任务参数
	var params map[string]interface{}
	if req.JobParam != "" {
		if err := json.Unmarshal([]byte(req.JobParam), &params); err != nil {
			return fmt.Errorf("failed to parse job params: %w", err)
		}
	}

	// 模拟数据导出处理
	steps := []string{
		"Connecting to database",
		"Querying data",
		"Processing data",
		"Generating export file",
		"Uploading to storage",
	}

	for i, step := range steps {
		log.Printf("Job %d - Step %d/%d: %s", req.JobID, i+1, len(steps), step)
		time.Sleep(2 * time.Second) // 模拟处理时间
	}

	log.Printf("Data export completed for job %d", req.JobID)
	return nil
}

func (h *BusinessHandler) processUserNotification(req *TaskRequest) error {
	log.Printf("Starting user notification for job %d", req.JobID)

	// 解析通知参数
	var params struct {
		UserIDs []int  `json:"user_ids"`
		Message string `json:"message"`
		Type    string `json:"type"`
	}

	if req.JobParam != "" {
		if err := json.Unmarshal([]byte(req.JobParam), &params); err != nil {
			return fmt.Errorf("failed to parse notification params: %w", err)
		}
	}

	// 模拟发送通知
	log.Printf("Sending %s notification to %d users: %s",
		params.Type, len(params.UserIDs), params.Message)

	for _, userID := range params.UserIDs {
		log.Printf("Sending notification to user %d", userID)
		time.Sleep(500 * time.Millisecond) // 模拟发送时间
	}

	log.Printf("User notification completed for job %d", req.JobID)
	return nil
}

func (h *BusinessHandler) processReportGeneration(req *TaskRequest) error {
	log.Printf("Starting report generation for job %d", req.JobID)

	// 解析报表参数
	var params struct {
		ReportType string `json:"report_type"`
		DateRange  string `json:"date_range"`
		Format     string `json:"format"`
	}

	if req.JobParam != "" {
		if err := json.Unmarshal([]byte(req.JobParam), &params); err != nil {
			return fmt.Errorf("failed to parse report params: %w", err)
		}
	}

	// 模拟报表生成
	steps := []string{
		"Gathering data",
		"Calculating metrics",
		"Generating charts",
		"Rendering report template",
		"Saving report file",
	}

	for i, step := range steps {
		log.Printf("Job %d - Step %d/%d: %s", req.JobID, i+1, len(steps), step)
		time.Sleep(1 * time.Second)
	}

	log.Printf("Report generation completed for job %d", req.JobID)
	return nil
}

func (h *BusinessHandler) writeSuccessResponse(w http.ResponseWriter, message string, data map[string]interface{}) {
	response := TaskResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	h.writeJSONResponse(w, http.StatusOK, response)
}

func (h *BusinessHandler) writeErrorResponse(w http.ResponseWriter, message string, err error) {
	response := TaskResponse{
		Success: false,
		Message: message,
	}
	if err != nil {
		response.Data = map[string]interface{}{
			"error": err.Error(),
		}
	}
	h.writeJSONResponse(w, http.StatusBadRequest, response)
}

func (h *BusinessHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// 心跳处理
func (h *BusinessHandler) HandleHeartbeat(w http.ResponseWriter, r *http.Request) {
	h.writeSuccessResponse(w, "Handler is alive", map[string]interface{}{
		"handler_name": h.handlerName,
		"version":      h.version,
		"timestamp":    time.Now().Unix(),
	})
}

// 注册处理器到调度器
func (h *BusinessHandler) registerToScheduler(schedulerURL string, tasks []TaskConfig) error {
	for _, task := range tasks {
		registryData := map[string]interface{}{
			"task_type":    task.TaskType,
			"handler_name": h.handlerName,
			"endpoint_url": task.EndpointURL,
			"handler_type": "http",
			"version":      h.version,
			"timeout":      30,
			"retry_count":  3,
			"description":  task.Description,
		}

		jsonData, err := json.Marshal(registryData)
		if err != nil {
			return fmt.Errorf("failed to marshal registry data: %w", err)
		}

		resp, err := http.Post(schedulerURL+"/admin/handler/register",
			"application/json",
			fmt.NewReader(string(jsonData)))
		if err != nil {
			return fmt.Errorf("failed to register handler %s: %w", task.TaskType, err)
		}
		resp.Body.Close()

		log.Printf("Registered handler: %s -> %s", task.TaskType, task.EndpointURL)
	}

	return nil
}

type TaskConfig struct {
	TaskType    string
	EndpointURL string
	Description string
}

func main() {
	handler := NewBusinessHandler("business-processor", "1.0.0")

	// 设置路由
	http.HandleFunc("/process/data_export", handler.HandleDataExport)
	http.HandleFunc("/process/user_notification", handler.HandleUserNotification)
	http.HandleFunc("/process/report_generation", handler.HandleReportGeneration)
	http.HandleFunc("/health", handler.HandleHeartbeat)

	port := 8080
	log.Printf("Starting business handler server on port %d", port)

	// 定义要注册的任务
	tasks := []TaskConfig{
		{
			TaskType:    "data_export",
			EndpointURL: fmt.Sprintf("http://localhost:%d/process/data_export", port),
			Description: "数据导出处理器",
		},
		{
			TaskType:    "user_notification",
			EndpointURL: fmt.Sprintf("http://localhost:%d/process/user_notification", port),
			Description: "用户通知处理器",
		},
		{
			TaskType:    "report_generation",
			EndpointURL: fmt.Sprintf("http://localhost:%d/process/report_generation", port),
			Description: "报表生成处理器",
		},
	}

	// 启动HTTP服务器
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待服务器启动
	time.Sleep(2 * time.Second)

	// 注册到调度器（假设调度器在9999端口）
	schedulerURL := "http://localhost:9999"
	if err := handler.registerToScheduler(schedulerURL, tasks); err != nil {
		log.Printf("Failed to register to scheduler: %v", err)
		log.Printf("Will continue running without scheduler registration")
	} else {
		log.Printf("Successfully registered all handlers to scheduler")
	}

	// 定期发送心跳
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// 这里可以发送心跳到调度器
			log.Printf("Handler %s is alive", handler.handlerName)
		}
	}()

	log.Printf("Business handler is ready to process tasks")
	select {} // 保持服务运行
}
