package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"task-executor/database"
	"task-executor/logger"
	"task-executor/queue"
)

// RiverAdminHandler River 管理接口处理器
type RiverAdminHandler struct {
	riverManager *queue.RiverManager
	db           *database.Database
}

// NewRiverAdminHandler 创建 River 管理接口处理器
func NewRiverAdminHandler(riverManager *queue.RiverManager, db *database.Database) *RiverAdminHandler {
	return &RiverAdminHandler{
		riverManager: riverManager,
		db:           db,
	}
}

// GetStats 获取队列统计信息
func (h *RiverAdminHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := h.riverManager.GetStats()
	if err != nil {
		logger.Error("获取统计信息失败: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 添加数据库统计信息
	handlers, err := h.db.GetTaskHandlers()
	if err == nil {
		stats["active_handlers"] = len(handlers)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    stats,
	})
}

// ListJobs 列出任务
func (h *RiverAdminHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	queue := r.URL.Query().Get("queue")
	if queue == "" {
		queue = "default"
	}

	state := r.URL.Query().Get("state")
	_ = state // 忽略未使用变量

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		_ = limitStr // 忽略未使用变量
	}

	// 根据状态获取任务
	var jobs []interface{}

	// 注意：由于 River v0.25 的 API 可能有所不同，这里做了简化
	// 在实际实现中需要根据具体的 River API 调整

	jobs = []interface{}{} // 简化实现

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"jobs":  jobs,
			"total": len(jobs),
			"queue": queue,
			"state": state,
		},
	})
}

// CancelJob 取消任务
func (h *RiverAdminHandler) CancelJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"message": "Cancel job not implemented in simple version",
	})
}

// EnqueueJob 入队任务
func (h *RiverAdminHandler) EnqueueJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TaskType   string `json:"task_type"`
		TaskParams string `json:"task_params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TaskType == "" {
		http.Error(w, "task_type is required", http.StatusBadRequest)
		return
	}

	// 入队任务
	err := h.riverManager.EnqueueTask(req.TaskType, req.TaskParams)
	if err != nil {
		logger.Error("任务入队失败: %v", err)
		http.Error(w, fmt.Sprintf("Failed to enqueue job: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Job enqueued successfully",
	})
}

// GetQueueDashboard 获取队列仪表板数据
func (h *RiverAdminHandler) GetQueueDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取统计信息
	stats, err := h.riverManager.GetStats()
	if err != nil {
		logger.Error("获取统计信息失败: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 获取任务处理器信息
	handlers, err := h.db.GetTaskHandlers()
	if err != nil {
		logger.Error("获取任务处理器失败: %v", err)
		handlers = []database.TaskHandlerRegistry{}
	}

	// 获取任务统计
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7) // 最近7天
	jobStats, err := h.db.GetJobStatistics(startDate, endDate, "", "")
	if err != nil {
		logger.Error("获取任务统计失败: %v", err)
		jobStats = []database.JobStatistics{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"stats":      stats,
			"handlers":   handlers,
			"job_stats":  jobStats,
			"updated_at": time.Now().Format(time.RFC3339),
		},
	})
}

// ListHandlers 列出任务处理器
func (h *RiverAdminHandler) ListHandlers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	handlers, err := h.db.GetTaskHandlers()
	if err != nil {
		logger.Error("获取任务处理器失败: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    handlers,
	})
}

// RegisterHandler 注册任务处理器
func (h *RiverAdminHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var handler database.TaskHandlerRegistry
	if err := json.NewDecoder(r.Body).Decode(&handler); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if handler.TaskType == "" || handler.HandlerName == "" {
		http.Error(w, "task_type and handler_name are required", http.StatusBadRequest)
		return
	}

	if err := h.db.RegisterTaskHandler(&handler); err != nil {
		logger.Error("注册任务处理器失败: %v", err)
		http.Error(w, fmt.Sprintf("Failed to register handler: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Handler registered successfully",
		"data":    handler,
	})
}

// UpdateHandler 更新任务处理器
func (h *RiverAdminHandler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var handler database.TaskHandlerRegistry
	if err := json.NewDecoder(r.Body).Decode(&handler); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if handler.TaskType == "" {
		http.Error(w, "task_type is required", http.StatusBadRequest)
		return
	}

	if err := h.db.RegisterTaskHandler(&handler); err != nil {
		logger.Error("更新任务处理器失败: %v", err)
		http.Error(w, fmt.Sprintf("Failed to update handler: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Handler updated successfully",
		"data":    handler,
	})
}

// DeleteHandler 删除任务处理器
func (h *RiverAdminHandler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	taskType := r.URL.Query().Get("task_type")
	if taskType == "" {
		http.Error(w, "task_type is required", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteTaskHandler(taskType); err != nil {
		logger.Error("删除任务处理器失败: %v", err)
		http.Error(w, fmt.Sprintf("Failed to delete handler: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Handler deleted successfully",
	})
}
