package queue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"xxljob-go-executor/database"
	"xxljob-go-executor/logger"
	"xxljob-go-executor/models"
)

// DynamicJobHandler 动态任务处理器接口
type DynamicJobHandler interface {
	models.JobHandler
	GetHandlerInfo() *database.TaskHandlerRegistry
}

// HTTPJobHandler HTTP远程任务处理器
type HTTPJobHandler struct {
	registry   *database.TaskHandlerRegistry
	httpClient *http.Client
}

// NewHTTPJobHandler 创建HTTP任务处理器
func NewHTTPJobHandler(registry *database.TaskHandlerRegistry) *HTTPJobHandler {
	timeout := time.Duration(registry.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &HTTPJobHandler{
		registry: registry,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Execute 执行任务
func (h *HTTPJobHandler) Execute(ctx *models.JobContext) error {
	// 构造请求数据
	requestData := map[string]interface{}{
		"job_id":       ctx.JobID,
		"job_param":    ctx.JobParam,
		"log_id":       ctx.LogID,
		"log_datetime": ctx.LogDateTime.Unix(),
		"task_type":    h.registry.TaskType,
		"version":      h.registry.Version,
	}

	// 如果有额外配置，合并到请求中
	if h.registry.ConfigData != "" {
		var configData map[string]interface{}
		if err := json.Unmarshal([]byte(h.registry.ConfigData), &configData); err == nil {
			requestData["config"] = configData
		}
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %w", err)
	}

	// 发送HTTP请求
	logger.Info("发送任务到远程处理器: %s, URL: %s", h.registry.TaskType, h.registry.EndpointURL)

	req, err := http.NewRequest("POST", h.registry.EndpointURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "XXLJob-Go-Executor/1.0")
	req.Header.Set("X-Task-Type", h.registry.TaskType)
	req.Header.Set("X-Handler-Version", h.registry.Version)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("任务执行失败，HTTP状态码: %d, 响应: %s", resp.StatusCode, string(responseBody))
	}

	// 解析响应
	var response TaskExecutionResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		// 如果无法解析为标准响应格式，检查是否是简单的成功响应
		if resp.StatusCode == http.StatusOK {
			logger.Info("任务执行完成，处理器: %s, 响应: %s", h.registry.TaskType, string(responseBody))
			return nil
		}
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查业务执行结果
	if !response.Success {
		return fmt.Errorf("任务执行失败: %s", response.Message)
	}

	logger.Info("任务执行成功，处理器: %s, 消息: %s", h.registry.TaskType, response.Message)
	return nil
}

// GetHandlerInfo 获取处理器信息
func (h *HTTPJobHandler) GetHandlerInfo() *database.TaskHandlerRegistry {
	return h.registry
}

// TaskExecutionResponse 任务执行响应
type TaskExecutionResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// LocalJobHandler 本地任务处理器适配器
type LocalJobHandler struct {
	registry *database.TaskHandlerRegistry
	handler  models.JobHandler
}

// NewLocalJobHandler 创建本地任务处理器适配器
func NewLocalJobHandler(registry *database.TaskHandlerRegistry, handler models.JobHandler) *LocalJobHandler {
	return &LocalJobHandler{
		registry: registry,
		handler:  handler,
	}
}

// Execute 执行任务
func (h *LocalJobHandler) Execute(ctx *models.JobContext) error {
	return h.handler.Execute(ctx)
}

// GetHandlerInfo 获取处理器信息
func (h *LocalJobHandler) GetHandlerInfo() *database.TaskHandlerRegistry {
	return h.registry
}
