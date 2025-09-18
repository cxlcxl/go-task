package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"xxljob-go-executor/database"
	"xxljob-go-executor/logger"
)

type AdminHandler struct {
	db           *database.Database
	handlerAdmin *HandlerAdminHandler
}

func NewAdminHandler(db *database.Database) *AdminHandler {
	return &AdminHandler{
		db:           db,
		handlerAdmin: NewHandlerAdminHandler(db),
	}
}

// QueueConfigRequest 队列配置请求结构
type QueueConfigRequest struct {
	QueueName    string `json:"queue_name"`
	TableName    string `json:"table_name"`
	ServerID     string `json:"server_id"`
	MaxWorkers   int    `json:"max_workers"`
	ScanInterval int    `json:"scan_interval"`
	Status       int    `json:"status"`
}

// CreateQueueConfig 创建队列配置
func (h *AdminHandler) CreateQueueConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	var req QueueConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	config := database.QueueConfig{
		QueueName:    req.QueueName,
		TableName:    req.TableName,
		ServerID:     req.ServerID,
		MaxWorkers:   req.MaxWorkers,
		ScanInterval: req.ScanInterval,
		Status:       req.Status,
	}

	if err := h.db.GetDB().Create(&config).Error; err != nil {
		logger.Error("创建队列配置失败: %v", err)
		h.writeError(w, "创建队列配置失败", http.StatusInternalServerError)
		return
	}

	h.writeSuccess(w, config)
	logger.Info("队列配置创建成功: %s", req.QueueName)
}

// UpdateQueueConfig 更新队列配置
func (h *AdminHandler) UpdateQueueConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		h.writeError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.writeError(w, "无效的ID", http.StatusBadRequest)
		return
	}

	var req QueueConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	config := database.QueueConfig{
		QueueName:    req.QueueName,
		TableName:    req.TableName,
		ServerID:     req.ServerID,
		MaxWorkers:   req.MaxWorkers,
		ScanInterval: req.ScanInterval,
		Status:       req.Status,
		UpdatedAt:    time.Now(),
	}

	if err := h.db.GetDB().Where("id = ?", id).Updates(&config).Error; err != nil {
		logger.Error("更新队列配置失败: %v", err)
		h.writeError(w, "更新队列配置失败", http.StatusInternalServerError)
		return
	}

	h.writeSuccess(w, config)
	logger.Info("队列配置更新成功: %s", req.QueueName)
}

// GetQueueConfigs 获取队列配置列表
func (h *AdminHandler) GetQueueConfigs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	var configs []database.QueueConfig
	if err := h.db.GetDB().Find(&configs).Error; err != nil {
		logger.Error("获取队列配置失败: %v", err)
		h.writeError(w, "获取队列配置失败", http.StatusInternalServerError)
		return
	}

	h.writeSuccess(w, configs)
}

// DeleteQueueConfig 删除队列配置
func (h *AdminHandler) DeleteQueueConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.writeError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.writeError(w, "无效的ID", http.StatusBadRequest)
		return
	}

	if err := h.db.GetDB().Delete(&database.QueueConfig{}, id).Error; err != nil {
		logger.Error("删除队列配置失败: %v", err)
		h.writeError(w, "删除队列配置失败", http.StatusInternalServerError)
		return
	}

	h.writeSuccess(w, map[string]string{"message": "删除成功"})
	logger.Info("队列配置删除成功: ID=%d", id)
}

// GetServerInfos 获取服务器信息
func (h *AdminHandler) GetServerInfos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	var servers []database.ServerInfo
	if err := h.db.GetDB().Find(&servers).Error; err != nil {
		logger.Error("获取服务器信息失败: %v", err)
		h.writeError(w, "获取服务器信息失败", http.StatusInternalServerError)
		return
	}

	h.writeSuccess(w, servers)
}

// TaskRequest 任务请求结构
type TaskRequest struct {
	TaskType     string                 `json:"task_type"`
	TaskParams   map[string]interface{} `json:"task_params"`
	ScheduleTime string                 `json:"schedule_time"`
	MaxRetries   int                    `json:"max_retries"`
}

// CreateTask 创建任务
func (h *AdminHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	tableName := r.URL.Query().Get("table")
	if tableName == "" {
		h.writeError(w, "缺少表名参数", http.StatusBadRequest)
		return
	}

	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	// 解析时间
	scheduleTime, err := time.Parse("2006-01-02 15:04:05", req.ScheduleTime)
	if err != nil {
		h.writeError(w, "无效的时间格式", http.StatusBadRequest)
		return
	}

	// 序列化任务参数
	paramsJSON, err := json.Marshal(req.TaskParams)
	if err != nil {
		h.writeError(w, "任务参数序列化失败", http.StatusBadRequest)
		return
	}

	maxRetries := req.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	// 插入任务
	query := `INSERT INTO ` + tableName + ` (task_type, task_params, schedule_time, max_retries) VALUES (?, ?, ?, ?)`
	if err := h.db.GetDB().Exec(query, req.TaskType, string(paramsJSON), scheduleTime, maxRetries).Error; err != nil {
		logger.Error("创建任务失败: %v", err)
		h.writeError(w, "创建任务失败", http.StatusInternalServerError)
		return
	}

	h.writeSuccess(w, map[string]string{"message": "任务创建成功"})
	logger.Info("任务创建成功: 表=%s, 类型=%s", tableName, req.TaskType)
}

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (h *AdminHandler) writeSuccess(w http.ResponseWriter, data interface{}) {
	response := Response{
		Code:    200,
		Message: "成功",
		Data:    data,
	}
	h.writeJSON(w, response)
}

func (h *AdminHandler) writeError(w http.ResponseWriter, message string, statusCode int) {
	response := Response{
		Code:    statusCode,
		Message: message,
	}
	w.WriteHeader(statusCode)
	h.writeJSON(w, response)
}

func (h *AdminHandler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(data)
}

// Handler management delegation methods

// RegisterHandler 注册处理器
func (h *AdminHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	h.handlerAdmin.RegisterHandler(w, r)
}

// ListHandlers 获取处理器列表
func (h *AdminHandler) ListHandlers(w http.ResponseWriter, r *http.Request) {
	h.handlerAdmin.ListHandlers(w, r)
}

// GetHandler 获取单个处理器信息
func (h *AdminHandler) GetHandler(w http.ResponseWriter, r *http.Request) {
	h.handlerAdmin.GetHandler(w, r)
}

// UpdateHandler 更新处理器信息
func (h *AdminHandler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	h.handlerAdmin.UpdateHandler(w, r)
}

// DeleteHandler 删除处理器
func (h *AdminHandler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	h.handlerAdmin.DeleteHandler(w, r)
}

// HeartbeatHandler 处理器心跳接口
func (h *AdminHandler) HeartbeatHandler(w http.ResponseWriter, r *http.Request) {
	h.handlerAdmin.HeartbeatHandler(w, r)
}
