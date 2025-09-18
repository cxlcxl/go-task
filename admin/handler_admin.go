package admin

import (
	"encoding/json"
	"net/http"

	"xxljob-go-executor/database"
	"xxljob-go-executor/logger"
)

// HandlerAdminHandler 处理器管理接口
type HandlerAdminHandler struct {
	db *database.Database
}

// NewHandlerAdminHandler 创建处理器管理处理器
func NewHandlerAdminHandler(db *database.Database) *HandlerAdminHandler {
	return &HandlerAdminHandler{
		db: db,
	}
}

// RegisterHandler 注册任务处理器
func (h *HandlerAdminHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var registry database.TaskHandlerRegistry
	if err := json.NewDecoder(r.Body).Decode(&registry); err != nil {
		h.writeErrorResponse(w, "Invalid request body", err)
		return
	}

	// 验证必填字段
	if registry.TaskType == "" || registry.HandlerName == "" {
		h.writeErrorResponse(w, "task_type and handler_name are required", nil)
		return
	}

	// 设置默认值
	if registry.HandlerType == "" {
		registry.HandlerType = database.HandlerTypeHTTP
	}
	if registry.Status == 0 {
		registry.Status = database.HandlerStatusEnabled
	}
	if registry.Timeout == 0 {
		registry.Timeout = 30
	}
	if registry.RetryCount == 0 {
		registry.RetryCount = 3
	}
	if registry.Version == "" {
		registry.Version = "1.0.0"
	}

	// 注册处理器
	if err := h.db.RegisterTaskHandler(&registry); err != nil {
		h.writeErrorResponse(w, "Failed to register handler", err)
		return
	}

	logger.Info("任务处理器注册成功: %s", registry.TaskType)
	h.writeSuccessResponse(w, map[string]interface{}{
		"task_type": registry.TaskType,
		"message":   "Handler registered successfully",
	})
}

// ListHandlers 获取处理器列表
func (h *HandlerAdminHandler) ListHandlers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	handlers, err := h.db.GetTaskHandlers()
	if err != nil {
		h.writeErrorResponse(w, "Failed to get handlers", err)
		return
	}

	h.writeSuccessResponse(w, map[string]interface{}{
		"handlers": handlers,
		"count":    len(handlers),
	})
}

// GetHandler 获取单个处理器信息
func (h *HandlerAdminHandler) GetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	taskType := r.URL.Query().Get("task_type")
	if taskType == "" {
		h.writeErrorResponse(w, "task_type parameter is required", nil)
		return
	}

	handler, err := h.db.GetTaskHandlerByType(taskType)
	if err != nil {
		h.writeErrorResponse(w, "Handler not found", err)
		return
	}

	h.writeSuccessResponse(w, handler)
}

// UpdateHandler 更新处理器信息
func (h *HandlerAdminHandler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	taskType := r.URL.Query().Get("task_type")
	if taskType == "" {
		h.writeErrorResponse(w, "task_type parameter is required", nil)
		return
	}

	// 获取现有处理器信息
	existing, err := h.db.GetTaskHandlerByType(taskType)
	if err != nil {
		h.writeErrorResponse(w, "Handler not found", err)
		return
	}

	// 解析更新数据
	var updateData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		h.writeErrorResponse(w, "Invalid request body", err)
		return
	}

	// 更新字段
	if val, ok := updateData["handler_name"]; ok {
		if str, ok := val.(string); ok {
			existing.HandlerName = str
		}
	}
	if val, ok := updateData["endpoint_url"]; ok {
		if str, ok := val.(string); ok {
			existing.EndpointURL = str
		}
	}
	if val, ok := updateData["version"]; ok {
		if str, ok := val.(string); ok {
			existing.Version = str
		}
	}
	if val, ok := updateData["timeout"]; ok {
		if f, ok := val.(float64); ok {
			existing.Timeout = int(f)
		}
	}
	if val, ok := updateData["status"]; ok {
		if f, ok := val.(float64); ok {
			existing.Status = int(f)
		}
	}
	if val, ok := updateData["config_data"]; ok {
		if str, ok := val.(string); ok {
			existing.ConfigData = str
		}
	}
	if val, ok := updateData["description"]; ok {
		if str, ok := val.(string); ok {
			existing.Description = str
		}
	}

	// 保存更新
	if err := h.db.RegisterTaskHandler(existing); err != nil {
		h.writeErrorResponse(w, "Failed to update handler", err)
		return
	}

	logger.Info("任务处理器更新成功: %s", taskType)
	h.writeSuccessResponse(w, map[string]interface{}{
		"task_type": taskType,
		"message":   "Handler updated successfully",
	})
}

// DeleteHandler 删除处理器
func (h *HandlerAdminHandler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	taskType := r.URL.Query().Get("task_type")
	if taskType == "" {
		h.writeErrorResponse(w, "task_type parameter is required", nil)
		return
	}

	if err := h.db.DeleteTaskHandler(taskType); err != nil {
		h.writeErrorResponse(w, "Failed to delete handler", err)
		return
	}

	logger.Info("任务处理器删除成功: %s", taskType)
	h.writeSuccessResponse(w, map[string]interface{}{
		"task_type": taskType,
		"message":   "Handler deleted successfully",
	})
}

// HeartbeatHandler 处理器心跳接口
func (h *HandlerAdminHandler) HeartbeatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var heartbeat struct {
		TaskType string `json:"task_type"`
		Version  string `json:"version"`
		Status   string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&heartbeat); err != nil {
		h.writeErrorResponse(w, "Invalid request body", err)
		return
	}

	if heartbeat.TaskType == "" {
		h.writeErrorResponse(w, "task_type is required", nil)
		return
	}

	if err := h.db.UpdateHandlerHeartbeat(heartbeat.TaskType); err != nil {
		h.writeErrorResponse(w, "Failed to update heartbeat", err)
		return
	}

	h.writeSuccessResponse(w, map[string]string{
		"message": "Heartbeat updated successfully",
	})
}

// writeSuccessResponse 写入成功响应
func (h *HandlerAdminHandler) writeSuccessResponse(w http.ResponseWriter, data interface{}) {
	response := map[string]interface{}{
		"success": true,
		"data":    data,
	}
	h.writeJSONResponse(w, http.StatusOK, response)
}

// writeErrorResponse 写入错误响应
func (h *HandlerAdminHandler) writeErrorResponse(w http.ResponseWriter, message string, err error) {
	response := map[string]interface{}{
		"success": false,
		"message": message,
	}
	if err != nil {
		response["error"] = err.Error()
		logger.Error("%s: %v", message, err)
	}
	h.writeJSONResponse(w, http.StatusBadRequest, response)
}

// writeJSONResponse 写入JSON响应
func (h *HandlerAdminHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
