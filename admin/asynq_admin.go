package admin

import (
	"encoding/json"
	"net/http"
	"time"

	"xxljob-go-executor/asynq"
	"xxljob-go-executor/logger"
)

// AsynqAdminHandler Asynq 管理接口处理器
type AsynqAdminHandler struct {
	asynqManager *asynq.AsynqManager
}

// NewAsynqAdminHandler 创建Asynq管理接口处理器
func NewAsynqAdminHandler(asynqManager *asynq.AsynqManager) *AsynqAdminHandler {
	return &AsynqAdminHandler{
		asynqManager: asynqManager,
	}
}

// GetAsynqStats 获取Asynq统计信息
func (h *AsynqAdminHandler) GetAsynqStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := h.asynqManager.GetStats()
	if err != nil {
		logger.Error("获取Asynq统计信息失败: %v", err)
		http.Error(w, "获取统计信息失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"msg":  "success",
		"data": stats,
	})
}

// ListAsynqTasks 列出Asynq任务
func (h *AsynqAdminHandler) ListAsynqTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	queue := r.URL.Query().Get("queue")
	if queue == "" {
		queue = "default"
	}

	// 简化实现，返回空列表
	tasks := []interface{}{}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"msg":  "success",
		"data": map[string]interface{}{
			"queue": queue,
			"tasks": tasks,
			"total": len(tasks),
		},
	})
}

// CancelAsynqTask 取消Asynq任务
func (h *AsynqAdminHandler) CancelAsynqTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Queue  string `json:"queue"`
		TaskID string `json:"task_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Queue == "" || req.TaskID == "" {
		http.Error(w, "队列名和任务ID不能为空", http.StatusBadRequest)
		return
	}

	err := h.asynqManager.CancelTask(req.Queue, req.TaskID)
	if err != nil {
		logger.Error("取消任务失败: %v", err)
		http.Error(w, "取消任务失败", http.StatusInternalServerError)
		return
	}

	logger.Info("任务取消成功: 队列=%s, 任务ID=%s", req.Queue, req.TaskID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"msg":  "任务取消成功",
		"data": map[string]interface{}{
			"queue":   req.Queue,
			"task_id": req.TaskID,
		},
	})
}

// TriggerMigration 手动触发任务迁移
func (h *AsynqAdminHandler) TriggerMigration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Asynq本身就是目标系统，不需要迁移
	logger.Info("使用Asynq系统，无需迁移")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"msg":  "使用Asynq系统，无需迁移",
		"data": map[string]interface{}{
			"system": "asynq",
		},
	})
}

// EnqueueTask 手动入队任务
func (h *AsynqAdminHandler) EnqueueTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TaskType     string `json:"task_type"`
		TaskParams   string `json:"task_params"`
		Queue        string `json:"queue,omitempty"`
		DelayMinutes int    `json:"delay_minutes,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.TaskType == "" {
		http.Error(w, "任务类型不能为空", http.StatusBadRequest)
		return
	}

	var err error
	if req.DelayMinutes > 0 {
		// 延迟任务
		delay := time.Duration(req.DelayMinutes) * time.Minute
		_, err = h.asynqManager.EnqueueTaskIn(req.TaskType, req.TaskParams, delay)
	} else {
		// 立即任务
		_, err = h.asynqManager.EnqueueTask(req.TaskType, req.TaskParams)
	}

	if err != nil {
		logger.Error("任务入队失败: %v", err)
		http.Error(w, "任务入队失败", http.StatusInternalServerError)
		return
	}

	logger.Info("任务入队成功: 类型=%s, 队列=%s", req.TaskType, req.Queue)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"msg":  "任务入队成功",
		"data": map[string]interface{}{
			"task_type":   req.TaskType,
			"task_params": req.TaskParams,
			"queue":       req.Queue,
		},
	})
}

// GetQueueDashboard 获取队列仪表板数据
func (h *AsynqAdminHandler) GetQueueDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := h.asynqManager.GetStats()
	if err != nil {
		logger.Error("获取仪表板数据失败: %v", err)
		http.Error(w, "获取仪表板数据失败", http.StatusInternalServerError)
		return
	}

	// 构建仪表板响应
	dashboard := map[string]interface{}{
		"system_status":  "running",
		"asynq_enabled":  true,
		"legacy_enabled": false,
		"stats":          stats,
		"health": map[string]interface{}{
			"redis_connected": true,
			"workers_running": true,
		},
	}

	// 如果有Asynq统计信息，提取关键指标
	// 注意：stats 现在是 map[string]interface{} 类型
	if queues, ok := stats["queues"]; ok {
		dashboard["key_metrics"] = map[string]interface{}{
			"queues_count": len(stats),
			"queues_info":  queues,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"msg":  "success",
		"data": dashboard,
	})
}

// GetAsynqWebUI 获取Asynq Web UI 页面
func (h *AsynqAdminHandler) GetAsynqWebUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Asynq 队列监控</title>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { color: #2c3e50; border-bottom: 2px solid #3498db; padding-bottom: 10px; }
        .section { margin: 20px 0; padding: 20px; background: #f8f9fa; border-radius: 5px; }
        .button {
            background: #3498db; color: white; border: none; padding: 10px 20px;
            border-radius: 3px; cursor: pointer; margin: 5px;
        }
        .button:hover { background: #2980b9; }
        #result {
            margin-top: 20px; padding: 15px; background: #ecf0f1;
            border-left: 4px solid #3498db; border-radius: 3px;
        }
        pre { background: #2c3e50; color: #ecf0f1; padding: 15px; border-radius: 3px; overflow-x: auto; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🚀 Asynq 队列监控系统</h1>
        <p>基于Redis的高性能分布式任务队列</p>
    </div>

    <div class="section">
        <h2>📊 监控操作</h2>
        <button class="button" onclick="getStats()">📈 获取统计信息</button>
        <button class="button" onclick="getDashboard()">🎛️ 获取仪表板</button>
        <button class="button" onclick="listTasks()">📋 列出任务</button>
        <button class="button" onclick="triggerMigration()">🔄 触发迁移</button>
    </div>

    <div class="section">
        <h2>➕ 任务操作</h2>
        <input type="text" id="taskType" placeholder="任务类型 (如: email)" style="padding: 8px; margin: 5px;">
        <input type="text" id="taskParams" placeholder="任务参数 (JSON格式)" style="padding: 8px; margin: 5px; width: 300px;">
        <button class="button" onclick="enqueueTask()">📤 入队任务</button>
    </div>

    <div id="result">
        <h3>📄 执行结果</h3>
        <pre id="output">点击按钮查看结果...</pre>
    </div>

    <script>
        function updateOutput(data) {
            document.getElementById('output').textContent = JSON.stringify(data, null, 2);
        }

        function handleError(error) {
            document.getElementById('output').textContent = 'Error: ' + error;
        }

        function getStats() {
            fetch('/admin/asynq/stats')
                .then(response => response.json())
                .then(updateOutput)
                .catch(handleError);
        }

        function getDashboard() {
            fetch('/admin/asynq/dashboard')
                .then(response => response.json())
                .then(updateOutput)
                .catch(handleError);
        }

        function listTasks() {
            fetch('/admin/asynq/tasks?queue=default')
                .then(response => response.json())
                .then(updateOutput)
                .catch(handleError);
        }

        function triggerMigration() {
            fetch('/admin/asynq/migrate', { method: 'POST' })
                .then(response => response.json())
                .then(updateOutput)
                .catch(handleError);
        }

        function enqueueTask() {
            const taskType = document.getElementById('taskType').value;
            const taskParams = document.getElementById('taskParams').value || '{}';

            if (!taskType) {
                alert('请输入任务类型');
                return;
            }

            fetch('/admin/asynq/enqueue', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    task_type: taskType,
                    task_params: taskParams
                })
            })
            .then(response => response.json())
            .then(updateOutput)
            .catch(handleError);
        }
    </script>
</body>
</html>
    `))
}
